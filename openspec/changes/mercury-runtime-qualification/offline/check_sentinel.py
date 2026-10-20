"""Exercise only the stdlib negative-control fragment, never the CLI or bridge."""
import pathlib, tempfile, json, subprocess, sys
assert sys.platform == 'darwin', 'this OS-policy regression requires macOS'
test_root = pathlib.Path('.ateam/mercury-contract').resolve()
assert test_root.is_relative_to(pathlib.Path.home().resolve()), 'cwd must be under the user home for the file-denial control'
test_root.mkdir(parents=True, exist_ok=True)
source_path=pathlib.Path(sys.argv[1]) if len(sys.argv)>1 else pathlib.Path(__file__).with_name('reproduce.py')
source=source_path.read_text()
fragment=source.split('# Negative controls use documentation-only external IPs and a benign local sentinel.\n',1)[1].split('(R/"proxy.yaml").write_text',1)[0]
with tempfile.TemporaryDirectory(dir=test_root,prefix='ownership-check-') as temporary:
    parent=pathlib.Path(temporary).resolve(); R=parent/'root'; R.mkdir(); OUT=R/'out';OUT.mkdir()
    existing=parent/'mercury-outside-sentinel'; existing.write_text('preserve-existing-content')
    before=set(parent.iterdir())
    policy='(version 1)\n(allow default)\n(deny network*)\n(deny file-write*)\n'
    policy+='(allow file-write* (subpath '+json.dumps(str(R))+') (literal "/dev/null"))\n'
    policy+='(deny file-read* (subpath '+json.dumps(str(pathlib.Path.home()))+'))\n(allow file-read-metadata (subpath '+json.dumps(str(pathlib.Path.home()))+'))\n'
    policy+='(allow file-read* (subpath '+json.dumps(str(R))+'))\n'
    (R/'runtime.sb').write_text(policy)
    exec(compile(fragment,'reproduce-negative-controls','exec'),{'R':R,'OUT':OUT,'subprocess':subprocess,'tempfile':tempfile,'pathlib':pathlib})
    assert existing.read_text()=='preserve-existing-content','prior sentinel content changed'
    assert set(parent.iterdir())==before,'owned sentinel leaked'
print('PASS: pre-existing sibling preserved; owned sentinel cleaned')
