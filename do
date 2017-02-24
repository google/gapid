DO_DIR="`dirname \"$0\"`"
DO_DIR="`( cd \"$DO_DIR\" && pwd )`"
export GOPATH="$DO_DIR/third_party:`( cd \"$DO_DIR/../../../../\" && pwd )`"

cd ${DO_DIR} && go run ./cmd/do/*.go "$@"
