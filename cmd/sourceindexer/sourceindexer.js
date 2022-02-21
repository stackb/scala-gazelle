/**
 * @fileoverview sourceindexer.js parses a list of source files and outputs a
 * JSON summary of top-level symbols to stdout.
 */
const fs = require('fs');
const readline = require('readline');
const { Console } = require('console');
const { parseSource } = require('scalameta-parsers');

const debug = false;
const delim = Buffer.from([0x00]);

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
         * A set of top-level objects, qualified by their package name.
         * @type {Set<string>}
         */
        this.topObjects = new Set();

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
        if (tree.error) {
            this.visitError(tree);
        } else {
            this.visitNode(tree);
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
            default:
                this.console.log('unhandled node type', node.type, this.filename);
                // printNode(node);
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
    }

    visitImport(node) {
        node.importers.forEach(importer => this.visitImporter(importer));
    }

    visitImporter(node) {
        const ref = this.parseName(node.ref);
        node.importees.forEach(importee => {
            switch (importee.type) {
                case 'Importee.Name':
                    const name = importee.name.value;
                    this.imports.add([ref, name].join('.'))
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
        this.topObjects.add(this.packageQualifiedName(name));
    }

    visitDefnClass(node) {
        const name = this.parseName(node.name);
        this.topClasses.add(this.packageQualifiedName(name));
        this.visitStats(node.stats)
    }

    visitDefnTrait(node) {
        const name = this.parseName(node.name);
        this.topTraits.add(this.packageQualifiedName(name));
        this.visitStats(node.stats)
    }

    toObject() {
        const obj = {
            filename: this.filename,
        };

        const maybeAssign = (set, prop) => {
            const list = Array.from(set);
            if (list.length) {
                list.sort();
                obj[prop] = list;
            }
        };

        maybeAssign(this.packages, 'packages');
        maybeAssign(this.imports, 'imports');
        maybeAssign(this.topClasses, 'classes');
        maybeAssign(this.topTraits, 'traits');
        maybeAssign(this.topObjects, 'objects');

        return obj;
    }

    /**
     * Pretty print a node json.
     * @param {Node} node 
     */
    printNode(node) {
        this.console.log(JSON.stringify(node, null, 2));
    }

    /**
     * Parses a "Ref" node to a string.
     * @param {Ref} ref 
     * @returns {string}
     */
    parseName(ref) {
        switch (ref.type) {
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
                printNode(ref);
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
            console.warn('error parsing', filename, e);
        }
    });

    return srcs;
}

function main() {
    const args = process.argv.slice(2);
    if (debug) {
        console.warn('usage: sourceindexer.js -o OUTPUT_FILE -l LABEL [INPUT_FILES]');
        console.warn('args:', args);
    }

    // the output to write to (only valid when not in server mode)
    let output = process.stdout;
    // label is the bazel label that contains the file we are parsing, so it can
    // be included in the result json
    let label = '';
    // repoRoot is the absolute path to the root.
    let repoRoot = '';
    // inputs is a list of input files
    const inputs = [];

    for (let i = 0; i < args.length; i++) {
        const arg = args[i];
        switch (arg) {
            case '-o':
                outputFile = args[i + 1];
                i++;
                break;
            case '-l':
                label = args[i + 1];
                i++;
                break;
            case '-d':
                repoRoot = args[i + 1];
                i++;
                break;
            default:
                inputs.push(arg);
        }
    }

    // if the user supplied a list of files on the command line, parse those in
    // batch.  Otherwise, wait on stdin for parse requests.
    if (inputs.length > 0) {
        const srcs = parse(inputs)
        const result = JSON.stringify({ label, srcs }, null, 2);
        fs.writeFileSync(output, result);
        if (debug) {
            console.warn(`Wrote ${output} (${result.length} bytes)`);
        }

    } else {
        console.warn('Waiting for parse requests from stdin...');

        const io = readline.createInterface(process.stdin, process.stdout, undefined, false);

        async function run() {
            for await (const line of nextRequest()) {
                // const request = JSON.parse(line);
                const srcs = parse([line]);
                // console.warn(`parse file: "${line}"`, srcs);
                process.stdout.write(JSON.stringify(srcs[0]));
                process.stdout.write(delim);

            }
        }

        run();
        // io.question
        // io.on('line', (line) => {
        //     console.warn('line event: ', JSON.stringify(line, null, 2));
        //     const filename = line.trim();
        //     console.warn(`parse file: "${filename}"`);
        //     const result = parse(label, [filename]);
        //     inputs.length = 0;
        //     console.warn(`parse result: ${result}`);

        //     io.write(result);
        //     io.write('\n');
        // }).on('close', () => {
        //     process.exit(0);
        // });

        // io.resume();
    }

}

main();

async function* nextRequest() {
    const rl = readline.createInterface({
        input: process.stdin,
        output: undefined,
    });

    try {
        for (; ;) {
            yield new Promise((resolve) => rl.question("", resolve));
        }
    } finally {
        rl.close();
    }
}
