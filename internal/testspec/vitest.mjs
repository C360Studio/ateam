import { createRequire } from 'node:module';
import { readFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const modifiers = new Set(['skip', 'only', 'todo', 'fails', 'concurrent', 'sequential']);
const factories = new Set(['each', 'for', 'skipIf', 'runIf']);

// Parse declarations rather than searching source text, so strings and callback
// comments cannot satisfy attribution. Imports are resolved by local symbols;
// arbitrary wrappers and dynamic alias construction are deliberately unsupported.
export function scanSource(file, source, root = process.cwd()) {
  const ts = createRequire(path.join(root, 'ui/package.json'))('typescript');
  const sf = ts.createSourceFile(file, source, ts.ScriptTarget.Latest, true);
  const host = {
    getSourceFile: name => name === file ? sf : undefined,
    getDefaultLibFileName: () => '', writeFile: () => {}, getCurrentDirectory: () => '',
    getDirectories: () => [], fileExists: name => name === file,
    readFile: name => name === file ? source : undefined,
    getCanonicalFileName: name => name, useCaseSensitiveFileNames: () => true,
    getNewLine: () => '\n'
  };
  const checker = ts.createProgram([file], { noLib: true, noResolve: true, allowJs: true }, host).getTypeChecker();
  const records = [];
  const tests = new Set();
  const namespaces = new Set();
  const consumed = new Set();
  const line = node => sf.getLineAndCharacterOfPosition(typeof node === 'number' ? node : node.getStart(sf)).line + 1;
  const error = (node, message) => records.push({ file, line: line(node), error: message });
  for (const diagnostic of sf.parseDiagnostics) error(diagnostic.start ?? 0, ts.flattenDiagnosticMessageText(diagnostic.messageText, '\n'));
  if (records.length) return records;
  for (const statement of sf.statements) {
    if (!ts.isImportDeclaration(statement) || statement.moduleSpecifier.text !== 'vitest') continue;
    const clause = statement.importClause;
    if (!clause) continue;
    if (clause.name) error(clause.name, 'unsupported default Vitest import; use named or namespace imports');
    const bindings = clause.namedBindings;
    if (bindings && ts.isNamespaceImport(bindings)) namespaces.add(checker.getSymbolAtLocation(bindings.name));
    if (bindings && ts.isNamedImports(bindings)) {
      for (const element of bindings.elements) {
        if (['it', 'test'].includes((element.propertyName ?? element.name).text)) tests.add(checker.getSymbolAtLocation(element.name));
      }
    }
  }
  function base(node) {
    if (!ts.isIdentifier(node)) return false;
    if (ts.isPropertyAccessExpression(node.parent) && node.parent.name === node) return false;
    const symbol = checker.getSymbolAtLocation(node);
    return tests.has(symbol) || (!symbol && ['it', 'test'].includes(node.text));
  }
  function chain(node) {
    if (base(node)) return { refs: [node], props: [], configured: false };
    if (ts.isPropertyAccessExpression(node)) {
      if (ts.isIdentifier(node.expression) && namespaces.has(checker.getSymbolAtLocation(node.expression)) && ['it', 'test'].includes(node.name.text)) {
        return { refs: [node.expression], props: [], configured: false };
      }
      const inner = chain(node.expression);
      if (inner) return { ...inner, props: [...inner.props, node.name.text], configured: false };
    }
    if (ts.isCallExpression(node) || ts.isTaggedTemplateExpression(node)) {
      const expression = ts.isCallExpression(node) ? node.expression : node.tag;
      const inner = chain(expression);
      if (inner && factories.has(inner.props.at(-1))) return { ...inner, configured: true };
    }
    return undefined;
  }
  function visit(node) {
    if (ts.isImportDeclaration(node)) return;
    if (ts.isCallExpression(node)) {
      if ((node.expression.kind === ts.SyntaxKind.ImportKeyword ||
          (ts.isIdentifier(node.expression) && node.expression.text === 'require')) &&
          node.arguments.some(arg => ts.isStringLiteralLike(arg) && arg.text === 'vitest')) {
        error(node, 'unsupported dynamic Vitest import; use static named or namespace imports');
      }
      const found = chain(node.expression);
      if (found) {
        // A factory invocation is part of the outer declaration, not a test.
        const factory = factories.has(found.props.at(-1)) && !found.configured;
        if (!factory) {
          found.refs.forEach(ref => consumed.add(ref));
          const unknown = found.props.find(prop => !modifiers.has(prop) && !factories.has(prop));
          if (unknown) error(node, `unsupported Vitest modifier: ${unknown}`);
          else {
            const title = node.arguments[0];
            const name = title && (ts.isStringLiteralLike(title) ? title.text : title.getText(sf));
            if (!name) error(node, 'test declaration requires a name');
            else {
              const ranges = ts.getLeadingCommentRanges(source, node.getFullStart()) ?? [];
              const comments = [];
              let nextStart = node.getStart(sf);
              for (const range of [...ranges].reverse()) {
                if (line(nextStart) - line(range.end) > 1) break;
                comments.unshift({ line: line(range.pos), text: source.slice(range.pos, range.end) });
                nextStart = range.pos;
              }
              records.push({ file, line: line(node), end: line(node.getEnd()), name, comments });
            }
          }
        }
      }
    }
    ts.forEachChild(node, visit);
  }
  visit(sf);
  function checkUses(node) {
    if (ts.isImportDeclaration(node)) return;
    if (ts.isIdentifier(node) && !consumed.has(node)) {
      if (base(node)) error(node, 'unsupported test alias or declaration; use a direct Vitest test call');
      else if (namespaces.has(checker.getSymbolAtLocation(node))) {
        const parent = node.parent;
        if (!ts.isPropertyAccessExpression(parent) || parent.expression !== node || ['it', 'test'].includes(parent.name.text)) {
          error(node, 'unsupported Vitest namespace use; use a direct .test or .it call');
        }
      }
    }
    ts.forEachChild(node, checkUses);
  }
  checkUses(sf);
  return records.sort((a, b) => a.line - b.line);
}

if (process.argv[1] === '-' || (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url))) {
  const [root, ...files] = process.argv.slice(2);
  // Missing parser installation is an infrastructure failure, not a source diagnostic.
  createRequire(path.join(root, 'ui/package.json'))('typescript');
  const records = files.flatMap(file => {
    try { return scanSource(file, readFileSync(path.join(root, file), 'utf8'), root); }
    catch (err) { return [{ file, line: 1, error: err.message }]; }
  });
  process.stdout.write(JSON.stringify(records));
}
