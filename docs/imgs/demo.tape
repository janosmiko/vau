Output demo.gif
Set FontSize 16
Set Width 1200
Set Height 600
Set Theme "Catppuccin Mocha"
Set Padding 20

Type "echo 'We are going to demonstrate how vau works with Vault.'"
Enter
Type "echo 'We will show/hide secret values, and move a secret to a different path.'"
Enter
Sleep 3s

Type "clear"
Enter

Type "export VAULT_ADDR='http://127.0.0.1:8200'"
Enter
Type "export VAULT_TOKEN='example token'"
Enter

Type "vau"
Sleep 1s
Enter
Sleep 1s
Type "j"
Sleep 1s
Type "l"
Sleep 1s
Type "v"
Sleep 2s
Type "v"
Sleep 1s
Type "V"
Sleep 2s
Type "V"
Sleep 1s
Type "x"
Sleep 2s
Type "h"
Sleep 1s
Type "k"
Sleep 1s
Type "l"
Sleep 1s
Type "p"
Sleep 1s
Type "j"
Sleep 1s
Type "v"
Sleep 5s
