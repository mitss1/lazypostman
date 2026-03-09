# VHS demo tape for LazyPostman
# Install vhs: go install github.com/charmbracelet/vhs@latest
# Run: vhs demo.tape

Output demo.gif

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 700
Set Theme "Catppuccin Mocha"
Set Padding 20

Type "./lazypostman testdata/sample.postman_collection.json testdata/sample.postman_environment.json"
Enter
Sleep 2s

# Expand Users folder
Type "l"
Sleep 500ms

# Navigate to "List Users"
Type "j"
Sleep 500ms

# Send the request
Enter
Sleep 3s

# Scroll down in response to show JSON
Type "	"
Sleep 300ms
Type "	"
Sleep 500ms
Type "j"
Sleep 300ms
Type "j"
Sleep 300ms
Type "j"
Sleep 300ms
Type "j"
Sleep 300ms
Type "j"
Sleep 500ms

# Go back to tree, navigate to Get User
Type "	"
Sleep 300ms
Type "j"
Sleep 500ms

# Send Get User request
Enter
Sleep 3s

# Switch to response panel to see result
Type "	"
Sleep 300ms
Type "	"
Sleep 1s

# Show response headers tab
Type "t"
Sleep 1s

# Back to body
Type "t"
Sleep 1s

# Go to tree, navigate to Create User (POST)
Type "	"
Sleep 300ms
Type "j"
Sleep 500ms

# Switch to request panel to see body
Type "	"
Sleep 300ms
Type "t"
Sleep 300ms
Type "t"
Sleep 1s

# Open environment panel
Type "	"
Sleep 300ms
Type "v"
Sleep 1.5s

# Close env panel
Type "v"
Sleep 500ms

# Show help
Type "?"
Sleep 1.5s

# Quit
Type "q"
Sleep 500ms
