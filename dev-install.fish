# Global Variable
set -gx PROJECT_DIR (realpath  (dirname (status --current-filename) ))

# Start Compiling
go build $PROJECT_DIR/cmd/xdg-which

# Installing
mv xdg-which ~/.local/bin/xdg-which
