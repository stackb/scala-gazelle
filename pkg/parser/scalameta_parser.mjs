/**
 * @fileoverview scalameta_parser.mjs parses a list of source files and outputs a
 * JSON summary of top-level symbols to stdout.
 */
import * as fs from 'node:fs';
import * as http from 'node:http';
import { Worker, parentPort, workerData, isMainThread } from 'node:worker_threads';
import { Console } from 'node:console';
import { parseSource } from 'scalameta-parsers';

const __filename = new URL('', import.meta.url).pathname;
const debug = false;
const wantNameTypes = false;

// enableNestedImports will capture imports not at the top-level.  This can be
// useful, but in-practive is often used to narrow an import already named at
// the top-level, which then must be suppressed with resolve directives.
const enableNestedImports = true;

class Scope {
    /**
     * Construct a scope having a possibly-undefined parent scope.
     * @param {string} name
     * @param {Scope|undefined} parent 
     */
    constructor(name, parent) {
        /**
         * @type {string}
         */
        this.name = name;
        /**
         * @type {Scope|undefined}
         */
        this.parent = parent;
        /**
         * Imports is a list of imports in the scope.
         * @type {!Set<string>}
         */
        this.imports = new Set();
        /**
         * Symbols is a mapping of known symbols that resolve to an import.
         * @type {!Map<string,string>|undefined}
         */
        this.symbols = new Map();
    }

    /**
     * Add the given import to import set and bubble it up to the parent. The
     * sym argument is the current name, whereas the imp is the full import
     * string. For example, imp='com.typesafe.scalalogging.LazyLogging' and
     * sym='LazyLogging'. For 'com.typesafe.scalalogging._', sym is undefined.
     * @param {string} imp
     * @param {?string} sym
     */
    addImport(imp, sym) {
        this.imports.add(imp);
        if (sym) {
            this.addSymbol(sym, imp);
        }
        if (this.parent) {
            this.parent.addImport(imp, sym);
        }
    }

    /**
     * Get the scope qualified name.
     * @returns {string}
     */
    qname() {
        const names = [];
        let current = this;
        while (current) {
            names.push(current.name);
            current = current.parent;
        }
        return names.join('.');
    }

    /**
     * Add the given symbol and its fully-qualified import name.
     * @param {string} sym
     * @param {string} imp
     */
    addSymbol(sym, imp) {
        this.symbols.set(sym, imp);
    }

    /**
     * resolveSymbol attempts to match the given symbol to the known list of
     * fully-qualified imports.  If not match, return original symbol.
     * @param {string} sym
     * @returns {string}
     */
    resolveSymbol(sym) {
        const imp = this.symbols.get(sym);
        if (imp) {
            return imp;
        }
        return sym;
    }
}

/**
 * ScalaFile parses a scala source file and aggregates symbols discovered
 * by walking the AST.
 */
class ScalaFile {
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
         * The raw parse tree.
         */
        this.tree;

        /**
         * The root scope.
         */
        this.root = new Scope('root');

        /**
         * The scope stack.
         */
        this.scopes = [this.root];

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

    addName(name) {
        switch (name) {
            case "-":
            case "->":
            case "::":
            case ":=":
            case "!":
            case "???":
            case "*":
            case "&&":
            case "+":
            case "+=":
            case "<":
            case "<=":
            case "==":
            case ">":
            case ">=":
            case "~>":
                return;
        }
        if (name.startsWith(".")) {
            return;
        }
        if (isAllLowerCaseName(name)) {
            return;
        }
        this.names.add(name);
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
        // this.printNode(tree);
        if (tree.error) {
            this.console.log('Parse error:', this.filename);
            this.visitError(tree);
            return;
        } else {
            this.tree = tree;
        }

        this.traverse(this.root, this.root.name, tree, undefined /* parent */, [], (propName, node, parent, stack) => {
            if (!node) {
                return false
            }
            if (enableNestedImports && node.type === 'Import') {
                this.visitImport(node);
                return false;
            }

            let name = this.parseName(node);
            if (name) {
                if (wantNameTypes) {
                    const type = this.stackTypeName(node, stack);
                    name = `${name}<${type}>`;
                }
                this.addName(name);
            }
            return true;
        });

        this.visitNode(tree);
    }

    /**
     * currentScope returns the top of the scope stack.
     * @returns {!Scope}
     */
    currentScope() {
        return this.scopes[this.scopes.length - 1];
    }

    /**
     * Push scope create a new scope and pushed it on the stack
     * @param {string} name An arbitrary name for the scope
     * @param {!Scope} parent The parent scope
     * @returns {Scope|undefined}
     */
    pushScope(name, parent) {
        const scope = new Scope(name, parent);
        this.scopes.push(scope);
        return scope;
    }

    /**
     * popScope removes the top of the scope stack
     * @returns {Scope|undefined}
     */
    popScope() {
        if (this.scopes.length === 1) {
            throw new Error(`must not pop root scope!`);
        }
        return this.scopes.pop();
    }

    /**
     * Traverse an object, calling filter on each key/value pair to know whether
     * to continue.  The stack contains all parent objects which have a '.type'
     * field.
     * The visit function is a callback that takes the current key/value from the object
     * and the stack.
     * @see https://micahjon.com/2020/simple-depth-first-search-with-object-entries/.
     * @param {Scope} scope
     * @param {string} prop
     * @param {object} obj
     * @param {object | undefined} parent
     * @param {Array<object>} stack
     * @param {function} visit
     */
    traverse(scope, prop, obj, parent, stack, visit) {
        if (typeof obj !== 'object' || obj === null) {
            return; // scalar values like numbers and strings...
        }

        if (obj.type) {
            stack.push(obj);
            scope = this.pushScope(prop, scope);
            parent = obj;
        }

        if (Array.isArray(obj)) {
            obj.forEach((value) => {
                if (visit(prop, value, parent, stack)) {
                    this.traverse(scope, prop, value, parent, stack, visit);
                }
            });
        } else {
            Object.entries(obj).forEach(([key, value]) => {
                if (visit(key, value, parent, stack)) {
                    this.traverse(scope, key, value, parent, stack, visit);
                }
            });
        }

        if (obj.type) {
            stack.pop();
            this.popScope();
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
            this.console.log('visitNode ' + node.type);
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
            case 'Pkg.Body':
                this.visitPkgBody(node);
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
                if (debug) {
                    this.console.log('unhandled visitNode type', node.type, this.filename);
                    this.printNode(node);
                }
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
        // note: this changed in 4.10.0.  The Pkg object added a .Body.
        // See https://github.com/scalacenter/scalafix/pull/2079 and https://github.com/scalameta/scalameta/issues/3913.
        this.visitNode(node.body);
        this.pkgs.pop();
    }

    visitPkgBody(node) {
        this.visitStats(node.stats);
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
        const scope = this.currentScope();
        node.importees.forEach(importee => {
            switch (importee.type) {
                case 'Importee.Name':
                    scope.addImport([ref, importee.name.value].join('.'), importee.name.value)
                    break;
                case 'Importee.Rename':
                    scope.addImport([ref, importee.name.value].join('.'), importee.name.value)
                    break;
                case 'Importee.Unimport':
                    // an unimport is specifically excluded from the scala
                    // import symbol table, but since it still implies an
                    // interaction with the package we go ahead and index it
                    // here.
                    scope.addImport([ref, importee.name.value].join('.'))
                    break;
                case 'Importee.Wildcard':
                    scope.addImport([ref, '_'].join('.'))
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
        this.visitStats(node.stats);
    }

    visitDefnTrait(node) {
        const name = this.parseName(node.name);
        const qName = this.packageQualifiedName(name);
        this.topTraits.add(qName);
        this.parseExtends('trait', qName, node);
        this.visitStats(node.stats);
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

    resolveExtends() {
        this.extendsMap.forEach((classlist) => {
            classlist.classes = classlist.classes.map(sym => this.root.resolveSymbol(sym));
        });
    }

    /**
     * 
     * @param {!ParseRequest} request The parse request
     * @returns {!File} object / JSON representation per file.proto.
     */
    toFile(request) {
        const obj = {
            filename: this.filename,
        };
        if (this.error) {
            obj.error = this.error;
        }
        if (request.wantParseTree) {
            obj.tree = JSON.stringify(this.tree, null, 4);
        }

        this.resolveExtends();

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
        maybeAssignList(this.root.imports, 'imports');
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

    stackTypeName(node, stack) {
        const names = [];
        for (let i = 0; i < stack.length; i++) {
            if (stack[i].type) {
                names.push(stack[i].type);
            }
        }
        if (node.type) {
            names.push(node.type);
        }
        return names.join('/');
    }

    /**
     * Parses a typed node to a string.
     * @param {Node} ref 
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
            case 'Term.Param': {
                if (ref.decltpe) {
                    return this.parseName(ref.decltpe);
                }
                break;
            }
            case 'Term.Select': {
                const names = [];
                if (ref.qual) {
                    names.push(this.parseName(ref.qual));
                }
                if (ref.name) {
                    names.push(this.parseName(ref.name));
                }
                return names.join('.');
            }
            case 'Type.Select': {
                const names = [];
                if (ref.qual) {
                    names.push(this.parseName(ref.qual));
                }
                if (ref.name) {
                    names.push(this.parseName(ref.name));
                }
                return names.join('.');
            }
        }
        if (debug && ref.type) {
            this.console.warn('parseName: unhandled ref type:', ref.type);
            this.printNode(ref);
        }
    }

}

/**
 * parseFile parses a single file.
 * 
 * @param {!ParseRequest} request The parse request
 * @param {string} filename The file to parse (relative or absolute)
 * @returns {!ScalaFile}
 */
function parseFile(request, filename) {
    try {
        const file = new ScalaFile(filename);
        file.parse();
        return file.toFile(request);
    } catch (e) {
        return {
            filename: filename,
            error: e.message,
        };
    }
}

/**
 * parseFiles takes a list of input files and returns a list of results
 * 
 * @param {!ParseRequest} request The parse request
 * @param {!Array<string>} inputs The list of files to parse (relative or absolute)
 * @returns {!Array<ScalaFile>}
 */
function parseFiles(request, inputs) {
    return inputs.map(filename => parseFile(request, filename));
}

/**
 * parse takes a list of input files and returns a list of .
 * 
 * @param {!ParseRequest} request The parse request
 * @param {!Array<string>} inputs The list of files to parse (relative or absolute)
 * @returns {!Promise<!Array<ScalaFile>>}
 */
async function parseFilesParallel(request, inputs) {
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
 * @param {{filenames: !Array<string>}} request
 * @returns Array<!Object>
 */
async function processJSONRequest(request) {
    if (!Array.isArray(request.filenames)) {
        throw new Error(`bad request: expected '{ "filenames": [LIST OF FILES TO PARSE] }', but filenames list was not present`);
    }

    let files = [];
    if (process.env.PARALLEL_MODE) {
        files = await parseFilesParallel(request, request.filenames);
    } else {
        files = parseFiles(request, request.filenames);
    }

    return { files };
}

const requestHandler = (req, res) => {
    if (req.method != 'POST') {
        res.writeHead(400, { "Content-type": "application/json" });
        res.end(JSON.stringify({ error: "this server only services POST requests" }));
        return;
    }

    let body = "";
    req.on('data', (chunk) => {
        body += chunk.toString();
    });

    req.on('end', async () => {
        try {
            const result = await processJSONRequest(JSON.parse(body));
            res.writeHead(200, { "Content-type": "application/json" });;
            res.end(JSON.stringify(result));
        } catch (err) {
            res.writeHead(500, { "Content-type": "application/json" });;
            res.end(JSON.stringify({ error: err.message }));
        }
    });
}

if (isMainThread) {
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

/**
 * Determine if the given string starts with a lowercase letter.
 *
 * @param {string} name
 * @returns {boolean}
 */
function isAllLowerCaseName(name) {
    const parts = name.split(".");
    for (const part of parts) {
        if (!isLowerCaseName(part)) {
            return false;
        }
    }
    return true;
}

/**
 * Determine if the given string starts with a lowercase letter.
 *
 * @param {string} name
 * @returns {boolean}
 */
function isLowerCaseName(name) {
    if (name.length === 0) {
        return false;
    }
    const first = name.charAt(0);
    if (first === first.toLowerCase() && first !== first.toUpperCase()) {
        return true;
    }
    return false;
}
