"""The home-denial control must refuse out-of-home or symlink-escaped test roots."""
import pathlib
import subprocess
import sys
import tempfile
import unittest

SCRIPT = pathlib.Path(sys.argv.pop(1)).resolve() if len(sys.argv) > 1 else pathlib.Path(__file__).with_name('check_sentinel.py').resolve()
MESSAGE = 'cwd must be under the user home for the file-denial control'


@unittest.skipUnless(sys.platform == 'darwin', 'macOS Seatbelt control')
class SentinelCwd(unittest.TestCase):
    def run_check(self, cwd):
        return subprocess.run([sys.executable, '-I', str(SCRIPT)], cwd=cwd,
                              capture_output=True, text=True, timeout=15)

    def test_outside_home_refused_before_policy(self):
        with tempfile.TemporaryDirectory() as outside:
            self.assertFalse(pathlib.Path(outside).resolve().is_relative_to(pathlib.Path.home().resolve()))
            result = self.run_check(outside)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn(MESSAGE, result.stderr)
            self.assertFalse((pathlib.Path(outside) / '.ateam').exists())

    def test_symlink_escape_refused(self):
        parent = pathlib.Path('.ateam').resolve()
        parent.mkdir(exist_ok=True)
        with tempfile.TemporaryDirectory(dir=parent) as inside, tempfile.TemporaryDirectory() as outside:
            root = pathlib.Path(inside)
            (root / '.ateam').symlink_to(outside, target_is_directory=True)
            result = self.run_check(root)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn(MESSAGE, result.stderr)
            self.assertEqual(list(pathlib.Path(outside).iterdir()), [])

    def test_documented_worktree_succeeds(self):
        result = self.run_check(pathlib.Path.cwd())
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn('pre-existing sibling preserved; owned sentinel cleaned', result.stdout)


if __name__ == '__main__':
    unittest.main(verbosity=2)
