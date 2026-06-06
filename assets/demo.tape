# Snagify README demo tape (Charmbracelet VHS).
# Regenerate with:  vhs assets/demo.tape   (run from the repo root,
# after `go build -o snagify .`). The hidden block builds a throwaway
# demo project under /tmp so no real environment data is recorded.

Output assets/demo.gif

Set Shell "bash"
Set FontSize 18
Set Width 1000
Set Height 660
Set Padding 20
Set Theme "Catppuccin Mocha"

Hide
Type `export PS1="$ "` Enter
Type `SNAG="$PWD/snagify"` Enter
Type `rm -rf /tmp/snagify-demo && mkdir -p /tmp/snagify-demo && cd /tmp/snagify-demo` Enter
Type `cp "$SNAG" ./snagify && export PATH="$PWD:$PATH"` Enter
Type `printf '{"name":"checkout-service"}' > package.json` Enter
Type `: > pom.xml` Enter
Type `printf 'DATABASE_URL=\nJWT_SECRET=\nREDIS_URL=\n' > .env.example` Enter
Type `printf 'DATABASE_URL=postgres://demo@localhost:5432/app\n' > .env` Enter
Type "clear" Enter
Show

Type "snagify init" Sleep 500ms Enter
Sleep 1500ms
Type "snagify check" Sleep 500ms Enter
Sleep 4500ms
