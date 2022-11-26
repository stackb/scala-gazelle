/**
 * @fileoverview sourceindexer.mjs parses a list of source files and outputs a
 * JSON summary of top-level symbols to stdout.
 */
import * as fs from 'node:fs';
import * as http from 'node:http';
import * as path from 'node:path';
import { Worker, isMainThread, parentPort, workerData } from 'node:worker_threads';
import { Console } from 'node:console';
import { parseSource } from 'scalameta-parsers';

const __filename = new URL('', import.meta.url).pathname;
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
         * Key is the package-qualified-name, value is a an object with a list
         * of names in the form { classes: !Array<string> }
         * @type {Map<string,{classes:Array<string>}>}
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
                        let classList = this.extendsMap.get(key);
                        if (!classList) {
                            classList = { classes: [] };
                            this.extendsMap.set(key, classList);
                        }
                        classList.classes.push(tpe);
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
 * parse takes a list of input files and returns a list of .
 * 
 * @param {string>} filename The file to parse (relative or absolute)
 * @returns {!ScalaSourceInfo}
 */
function parseFile(filename) {
    const start = new Date().getTime();
    try {
        const src = new ScalaSourceFile(filename);
        src.parse();
        const result = src.toObject();
        result.elapsedMillis = new Date().getTime() - start;
        return result;
    } catch (e) {
        return {
            filename: filename,
            error: e.message,
        };
    }
}

/**
 * parse takes a list of input files and returns a list of .
 * 
 * @param {!Array<string>} inputs The list of files to parse (relative or absolute)
 * @returns {!Array<ScalaSourceInfo>}
 */
async function parseFiles(inputs) {
    return inputs.map(parseFile);
}

/**
 * parse takes a list of input files and returns a list of .
 * 
 * @param {!Array<string>} inputs The list of files to parse (relative or absolute)
 * @returns {!Array<ScalaSourceInfo>}
 */
async function parseFilesParallel(inputs) {
    const work = inputs.map(filename => {
        return new Promise((resolve, reject) => {
            const worker = new Worker(__filename, { workerData: filename });
            worker.on('message', resolve);
            worker.on('error', reject);
        });
    });
    return Promise.all(work);
}

/**
 * Process a parse request
 * @param {{files: !Array<string>}} request 
 * @returns Array<!Object>
 */
async function processJSONRequest(request) {
    if (!Array.isArray(request.files)) {
        throw new Error(`bad request: expected '{ "files": [LIST OF FILES TO PARSE] }', but files list was not present`);
    }

    const start = new Date().getTime();
    let scalaFiles = [];
    if (process.env.PARALLEL_MODE) {
        scalaFiles = await parseFilesParallel(request.files);
    } else {
        scalaFiles = await parseFiles(request.files);
    }
    const elapsedMillis = new Date().getTime() - start;

    return { scalaFiles, elapsedMillis };
}

function processApplicationJSON(data) {
    return processJSONRequest(JSON.parse(data));
}

const requestHandler = (req, res) => {
    if (req.method != 'POST') {
        res.writeHead(400, { "Content-type": "application/json" });
        res.end(JSON.stringify({ error: "this server only services POST requests" }));
        return;
    }

    const data = [];
    req.on('data', (chunk) => {
        data.push(chunk);
    });

    req.on('end', async () => {
        try {
            const result = await processApplicationJSON(data);
            res.writeHead(200, { "Content-type": "application/json" });;
            res.end(JSON.stringify(result));
        } catch (err) {
            res.writeHead(500, { "Content-type": "application/json" });;
            res.end(JSON.stringify({ error: err.message }));
        }
    });
}

if (true) {
    const server = http.createServer(requestHandler)
    const port = process.env.PORT || 3000;
    server.listen(port, (err) => {
        if (err) {
            return console.log('something bad happened', err)
        }
        if (debug) {
            console.log(`server is listening on ${port} (${__filename})`);
        }
    });
} else {
    const filename = workerData;
    const result = parseFile(filename);
    parentPort.postMessage(result);
}
