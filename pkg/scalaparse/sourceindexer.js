/**
 * @fileoverview sourceindexer.js parses a list of source files and outputs a
 * JSON summary of top-level symbols to stdout.
 */
const fs = require('fs');
import { serve } from "bun";
const { Console } = require('console');
const { parseSource } = require('scalameta-parsers');

const version = "1.0.0";
const debug = false;
const delim = Buffer.from([0x00]);
// enableNestedImports will capture imports not at the top-level.  This can be
// useful, but in-practive is often used to narrow an import already named at
// the top-level, which then must be suppressed with resolve directives.
const enableNestedImports = false;

/**
 * ScalaSourceFile parses a scala source file and aggregates symbols discovered
 * by walking the AST.
 */
class ScalaSourceFile {
    constructor(filename) {
        /**
         * a console that always prints to stderr.
         */
        this.console = new Console(process.stderr, process.stderr);

        /**
         * The current source filename.
         */
        this.filename = filename;

        /**
         * The stack of package names.  This is used to resolve package
         * membership when visiting top-level objects and classes.
         * @type {Array<string>}
         */
        this.pkgs = [];

        /**
         * A set of packages in the file (e.g. 'org.scalameta').
         * @type {Set<string>}
         */
        this.packages = new Set();

        /**
         * A set of package names that exist in the source.
         * @type {Set<string>}
         */
        this.imports = new Set();

        /**
         * An error, if the tree failed to parse.
         * @type {string|undefined}
         */
        this.error = undefined;

        /**
         * A set of top-level objects, qualified by their package name.
         * @type {Set<string>}
         */
        this.topObjects = new Set();

        /**
         * A set of top-level values, qualified by their package name.
         * @type {Set<string>}
         */
        this.topVals = new Set();

        /**
         * A set of top-level types, qualified by their package name.
         * @type {Set<string>}
         */
        this.topTypes = new Set();

        /**
         * A set of top-level classes, qualified by their package name.
         * @type {Set<string>}
         */
        this.topClasses = new Set();

        /**
         * A set of top-level traits, qualified by their package name.
         * @type {Set<string>}
         */
        this.topTraits = new Set();

        /**
         * A set of names anywhere in the file.
         * @type {Set<string>}
         */
        this.names = new Set();

        /**
         * If type, trait, or class extends another symbol, record that here.
         * Key is the package-qualified-name, value is a list of names.
         * @type {Map<string,Array<string>>}
         */
        this.extendsMap = new Map();
    }

    /**
     * Runs the parse.
     */
    parse() {
        if (debug) {
            this.console.log('Parsing', this.filename);
        }
        const buffer = fs.readFileSync(this.filename);
        const tree = parseSource(buffer.toString());
        //this.printNode(tree);

        this.traverse(tree, [], (key, node, stack) => {
            if (!node) {
                return false
            }
            if (node.type === 'Term.Name' && node.value) {
                this.names.add(node.value);
            }
            if (enableNestedImports) {
                if (node.type === 'Import') {
                    this.visitImport(node);
                    return false;
                }
            }
            return true;
        });

        if (tree.error) {
            this.visitError(tree);
        } else {
            this.visitNode(tree);
        }
    }

    /**
     * Traverse an object, calling filter on each key/value pair to know whether
     * to continue.  The stack contains all parent objects which have a '.type'
     * field.
     * @see https://micahjon.com/2020/simple-depth-first-search-with-object-entries/.
     * @param  {object} obj
     * @param  {Array<object>} stack
     * @param  {function} filter
     */
    traverse(obj, stack, filter) {
        if (typeof obj !== 'object' || obj === null) {
            return;
        }
        if (obj.type) {
            stack.push(obj);
        }
        Object.entries(obj).forEach(([key, value]) => {
            // Key is either an array index or object key
            if (filter(key, value, stack)) {
                this.traverse(value, stack, filter);
            }
        });
        if (obj.type) {
            stack.pop();
        }
    }

    /**
     * packageQualifiedName returns the dotted name corresponding to the current
     * package nesting stack.
     * @param {string} name 
     * @returns 
     */
    packageQualifiedName(name) {
        const names = this.pkgs.slice(0);
        names.push(name);
        return names.join('.');
    }

    visitError(node) {
        this.error = node.error;
        this.printNode(node);
    }

    visitNode(node) {
        if (debug) {
            this.console.log('visit ' + node.type);
        }
        switch (node.type) {
            case 'Source':
                this.visitSource(node);
                break;
            case 'Pkg':
                this.visitPkg(node);
                break;
            case 'Import':
                this.visitImport(node);
                break;
            case 'Pkg.Object':
                this.visitPkgObject(node);
                break;
            case 'Defn.Object':
                this.visitDefnObject(node);
                break;
            case 'Defn.Class':
                this.visitDefnClass(node);
                break;
            case 'Defn.Trait':
                this.visitDefnTrait(node);
                break;
            case 'Defn.Val':
                this.visitDefnVal(node);
                break;
            case 'Defn.Type':
                this.visitDefnType(node);
                break;
            case 'Template':
                this.visitTemplate(node);
                break;
            default:
                this.console.log('unhandled node type', node.type, this.filename);
                this.printNode(node);
                this.visitStats(node.stats);
        }
    }

    visitStats(stats) {
        if (stats) {
            for (const child of stats) {
                this.visitNode(child);
            }
        }
    }

    visitSource(node) {
        this.visitStats(node.stats);
    }

    visitPkg(node) {
        const name = this.parseName(node.ref);
        this.packages.add(this.packageQualifiedName(name));
        this.pkgs.push(name);
        this.visitStats(node.stats);
        this.pkgs.pop();
    }

    visitPkgObject(node) {
        const name = this.parseName(node.name);
        this.topObjects.add(this.packageQualifiedName(name));
        this.packages.add(this.packageQualifiedName(name));

        this.pkgs.push(name);
        this.visitNode(node.templ);
        this.pkgs.pop();
    }

    visitTemplate(node) {
        this.visitStats(node.stats);
    }

    visitImport(node) {
        node.importers.forEach(importer => this.visitImporter(importer));
    }

    visitImporter(node) {
        const ref = this.parseName(node.ref);
        node.importees.forEach(importee => {
            switch (importee.type) {
                case 'Importee.Name':
                    this.imports.add([ref, importee.name.value].join('.'))
                    break;
                case 'Importee.Rename':
                    this.imports.add([ref, importee.name.value].join('.'))
                    break;
                case 'Importee.Unimport':
                    // an unimport is specifically excluded from the scala
                    // import symbol table, but since it still implies an
                    // interaction with the package we go ahead and index it
                    // here.
                    this.imports.add([ref, importee.name.value].join('.'))
                    break;
                case 'Importee.Wildcard':
                    this.imports.add([ref, '_'].join('.'))
                    break;
                default:
                    this.console.log('unhandled importee type', importee.type);
            }
        });
    }

    visitDefnObject(node) {
        const name = this.parseName(node.name);
        const qName = this.packageQualifiedName(name);
        this.topObjects.add(qName);
        this.parseExtends('object', qName, node);
    }

    visitDefnClass(node) {
        const name = this.parseName(node.name);
        const qName = this.packageQualifiedName(name);
        this.topClasses.add(qName);
        this.parseExtends('class', qName, node);
        this.visitStats(node.stats)
    }

    visitDefnTrait(node) {
        const name = this.parseName(node.name);
        const qName = this.packageQualifiedName(name);
        this.topTraits.add(qName);
        this.parseExtends('trait', qName, node);
        this.visitStats(node.stats)
    }

    visitDefnVal(node) {
        // TODO(pcj): what are the reasonable vars to record?
        if (Array.isArray(node.pats) && node.pats.length && node.pats[0].type == "Pat.Var" && node.pats[0].name) {
            const name = this.parseName(node.pats[0].name);
            this.topVals.add(this.packageQualifiedName(name));
        }
    }

    visitDefnType(node) {
        const name = this.parseName(node.name);
        this.topTypes.add(this.packageQualifiedName(name));
    }

    parseExtends(type, qName, node) {
        const key = `${type} ${qName}`;
        if (node.templ) {
            for (const init of node.templ.inits) {
                // this.printNode(init);
                if (init.tpe) {
                    const tpe = this.parseName(init.tpe);
                    if (tpe) {
                        let symbols = this.extendsMap.get(key);
                        if (!symbols) {
                            symbols = [];
                            this.extendsMap.set(key, symbols);
                        }
                        symbols.push(tpe);
                    }
                }
            }
        }
    }

    toObject() {
        const obj = {
            filename: this.filename,
        };
        if (this.error) {
            obj.error = this.error;
        }

        const maybeAssignList = (set, prop) => {
            const list = Array.from(set);
            if (list.length) {
                list.sort();
                obj[prop] = list;
            }
        };

        const maybeAssignMap = (map, prop) => {
            if (!map.size) {
                return;
            }
            let m = Object.create(null);
            for (let [k, v] of map) {
                m[k] = v;
            }
            obj[prop] = m;
        };

        maybeAssignList(this.packages, 'packages');
        maybeAssignList(this.imports, 'imports');
        maybeAssignList(this.topClasses, 'classes');
        maybeAssignList(this.topTraits, 'traits');
        maybeAssignList(this.topObjects, 'objects');
        maybeAssignList(this.topVals, 'vals');
        maybeAssignList(this.topTypes, 'types');
        maybeAssignList(this.names, 'names');
        maybeAssignMap(this.extendsMap, 'extends');

        return obj;
    }

    /**
     * Pretty print a node json.
     * @param {Node} node 
     */
    printNode(node) {
        this.console.warn(JSON.stringify(node, null, 2));
    }

    /**
     * Parses a "Ref" node to a string.
     * @param {Ref} ref 
     * @returns {string}
     */
    parseName(ref) {
        switch (ref.type) {
            case 'Type.Apply':
                return this.parseName(ref.tpe);
            case 'Type.Name':
                return ref.value;
            case 'Term.Name':
                return ref.value;
            case 'Term.Select':
                const names = [];
                if (ref.qual) {
                    names.push(this.parseName(ref.qual));
                }
                if (ref.name) {
                    names.push(this.parseName(ref.name));
                }
                return names.join('.');
            default:
                this.console.warn('unhandled ref type:', ref.type);
                this.printNode(ref);
        }
    }

}

/**
 * parse takes a list of input files and write a JSON.
 * 
 * @param {!Array<string>} inputs The list of files to parse (relative or absolute)
 * @returns {!Array<ScalaSourceInfo>}
 */
function parse(inputs) {
    const srcs = [];
    inputs.forEach(filename => {
        try {
            const src = new ScalaSourceFile(filename);
            src.parse();
            srcs.push(src.toObject());
        } catch (e) {
            srcs.push({
                filename: filename,
                error: e.message,
            });
        }
    });

    return srcs;
}

const server = serve({
    port: process.env.PORT || 3000,
    fetch(request) {
        return new Response("Welcome to Bun!");
    },
});

// Stop the server after 5 seconds
setTimeout(() => {
    server.stop();
}, 5000);


// function main() {
//     const args = process.argv.slice(2);

//     // label is the bazel label that contains the file we are parsing, so it can
//     // be included in the result json
//     let label = '';
//     // repoRoot is the absolute path to the root.
//     let repoRoot = '';
//     // inputs is a list of input files
//     const inputs = [];

//     for (let i = 0; i < args.length; i++) {
//         const arg = args[i];
//         switch (arg) {
//             case '-l':
//                 label = args[i + 1];
//                 i++;
//                 break;
//             case '-d':
//                 repoRoot = args[i + 1];
//                 i++;
//                 break;
//             case '--version':
//                 process.stdout.write(`${version}\n`);
//                 process.exit(0);
//             default:
//                 inputs.push(arg);
//         }
//     }



//     if (inputs.length > 0) {
//         // if the user supplied a list of files on the command line, parse those in
//         // batch.
//         const srcs = parse(inputs)
//         const result = JSON.stringify({ label, srcs }, null, 2);
//         process.stdout.write(result);
//         if (debug) {
//             console.warn(`Wrote ${output} (${result.length} bytes)`);
//         }
//     } else {
//         // otherwise, wait on stdin for parse requests and write NUL delimited json messages.
//         // exit when we see a request line 'EXIT'
//         async function run() {
//             for await (const line of nextRequest()) {
//                 if (line === 'EXIT') {
//                     process.exit(0);
//                 }
//                 const srcs = parse([line]);
//                 // console.warn(`parse file: "${line}"`, srcs);
//                 process.stdout.write(JSON.stringify(srcs[0]));
//                 process.stdout.write(delim);
//             }
//         }
//         run();
//     }

// }

// main();

// async function* nextRequest() {
//     const rl = readline.createInterface({
//         input: process.stdin,
//         output: undefined,
//     });

//     try {
//         for (; ;) {
//             yield new Promise((resolve) => rl.question("", resolve));
//         }
//     } finally {
//         rl.close();
//     }
// }
