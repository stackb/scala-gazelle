/**
 * @fileoverview sourceindexer.js parses a list of source files and outputs a
 * JSON summary of top-level symbols to stdout.
 */
const fs = require('fs');
const { Console } = require('console');
const { parseSource } = require('scalameta-parsers');

const debug = false;

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

function main() {
    const args = process.argv.slice(2);
    if (debug) {
        console.warn('args:', args);
    }

    if (args.length < 5) {
        console.warn('problem: -o OUTPUT_FILE, -l LABEL, and at least one source file must be supplied.');
        console.error('usage: sourceindexer.js -o OUTPUT_FILE -l LABEL [INPUT_FILES]');
    }

    let outputFile = '/dev/stdout'
    let label = '';
    const inputs = [];

    for (let i = 0; i < args.length; i++) {
        const arg = args[i];
        if (arg === '-o') {
            outputFile = args[i + 1];
            i++;
            continue;
        } else if (arg === '-l') {
            label = args[i + 1];
            i++;
            continue;
        }
        inputs.push(arg);
    }

    const srcs = [];
    inputs.forEach(filename => {
        try {
            const src = new ScalaSourceFile(filename);
            src.parse();
            srcs.push(src.toObject());
        } catch (e) {
            console.warn('error parsing', filename, e);
        }
    });

    const data = JSON.stringify({ label, srcs }, null, 2);
    fs.writeFileSync(outputFile, data);

    if (debug) {
        console.warn(`Wrote ${outputFile} (${data.length} bytes)`);
    }
}

main();