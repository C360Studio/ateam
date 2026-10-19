// Offline plumbing demonstration. Fixed reference sources are not agent performance evidence.
let input = '';
process.stdin.setEncoding('utf8');
process.stdin.on('data', chunk => { input += chunk; });
process.stdin.on('end', () => {
  const request = JSON.parse(input);
  const sources = {
    'safe-path': `package safepath
import ("fmt"; "path/filepath"; "strings")
func Resolve(base, rel string) (string, error) {
 b, err := filepath.Abs(base); if err != nil { return "", err }
 p := filepath.Join(b, rel)
 r, err := filepath.Rel(b, p)
 if err != nil || r == ".." || strings.HasPrefix(r, ".."+string(filepath.Separator)) { return "", fmt.Errorf("outside base") }
 return p, nil
}`,
    'sql-user': `package sqluser
import "database/sql"
func FindByEmail(db *sql.DB, email string) (int64, error) {
 var id int64
 err := db.QueryRow("SELECT id FROM users WHERE email = ?", email).Scan(&id)
 return id, err
}`,
    'gitea-webhook': `package webhook
import ("crypto/hmac"; "crypto/sha256"; "encoding/hex")
func VerifyGiteaSignature(payload, secret []byte, sigHeader string) bool {
 if len(secret) == 0 || sigHeader == "" { return false }
 signature, err := hex.DecodeString(sigHeader); if err != nil { return false }
 mac := hmac.New(sha256.New, secret); mac.Write(payload)
 return hmac.Equal(mac.Sum(nil), signature)
}`,
  };
  if (request.version !== 1 || !sources[request.trial.task]) process.exit(1);
  process.stdout.write(JSON.stringify({ version: 1, source: sources[request.trial.task], usage: null }));
});
