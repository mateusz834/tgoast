#/usr/bin/env bash
set -e

BRANCH="go1.24.0"
TMP_DIR=$(mktemp -d --suffix "gorepo")
SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

set -x

git clone https://github.com/golang/go.git "$TMP_DIR"
cd "$TMP_DIR"

git filter-repo \
    --path-regex "^src/go/doc/(?!comment).*" \
    --path src/go/ast \
    --path src/go/constant \
    --path src/go/format \
    --path src/go/importer \
    --path src/go/internal/gccgoimporter \
    --path src/go/internal/gcimporter \
    --path src/go/internal/srcimporter \
    --path src/go/internal/typeparams \
    --path src/go/parser \
    --path src/go/printer \
    --path src/go/scanner \
    --path src/go/token \
    --path src/go/types \
    --path src/internal/bisect \
    --path src/internal/buildcfg \
    --path src/internal/cfg \
    --path src/internal/diff \
    --path src/internal/goarch \
    --path src/internal/godebug \
    --path src/internal/godebugs \
    --path src/internal/goexperiment \
    --path src/internal/goversion \
    --path src/internal/lazyregexp \
    --path src/internal/pkgbits \
    --path src/internal/platform \
    --path src/internal/race \
    --path src/internal/saferio \
    --path src/internal/testenv \
    --path src/internal/txtar \
    --path src/internal/types/errors \
    --path src/internal/types/testdata/check \
    --path src/internal/types/testdata/examples \
    --path src/internal/types/testdata/fixedbugs \
    --path src/internal/types/testdata/spec \
    --path src/internal/xcoff \
    --path src/internal/exportdata \
    --path-rename src/go/ast:ast \
    --path-rename src/go/constant:constant \
    --path-rename src/go/doc:doc \
    --path-rename src/go/format:format \
    --path-rename src/go/importer:importer \
    --path-rename src/go/internal/gccgoimporter:internal/go/gccgoimporter \
    --path-rename src/go/internal/gcimporter:internal/go/gcimporter \
    --path-rename src/go/internal/srcimporter:internal/go/srcimporter \
    --path-rename src/go/internal/typeparams:internal/go/typeparams \
    --path-rename src/go/parser:parser \
    --path-rename src/go/printer:printer \
    --path-rename src/go/scanner:scanner \
    --path-rename src/go/token:token \
    --path-rename src/go/types:types \
    --path-rename src/internal/bisect:internal/bisect \
    --path-rename src/internal/buildcfg:internal/buildcfg \
    --path-rename src/internal/cfg:internal/cfg \
    --path-rename src/internal/diff:internal/diff \
    --path-rename src/internal/goarch:internal/goarch \
    --path-rename src/internal/godebug:internal/godebug \
    --path-rename src/internal/godebugs:internal/godebugs \
    --path-rename src/internal/goexperiment:internal/goexperiment \
    --path-rename src/internal/goversion:internal/goversion \
    --path-rename src/internal/lazyregexp:internal/lazyregexp \
    --path-rename src/internal/pkgbits:internal/pkgbits \
    --path-rename src/internal/platform:internal/platform \
    --path-rename src/internal/race:internal/race \
    --path-rename src/internal/saferio:internal/saferio \
    --path-rename src/internal/testenv:internal/testenv \
    --path-rename src/internal/txtar:internal/txtar \
    --path-rename src/internal/types/errors:internal/types/errors \
    --path-rename src/internal/types/testdata/check:internal/types/testdata/check \
    --path-rename src/internal/types/testdata/examples:internal/types/testdata/examples \
    --path-rename src/internal/types/testdata/fixedbugs:internal/types/testdata/fixedbugs \
    --path-rename src/internal/types/testdata/spec:internal/types/testdata/spec \
    --path-rename src/internal/xcoff:internal/xcoff \
    --path-rename src/internal/exportdata:internal/exportdata \

cd "$SCRIPT_DIR/.."

git fetch --no-tags "$TMP_DIR" $BRANCH:upstream-$BRANCH

set +e
git merge upstream-$BRANCH
set -e

rm internal/buildcfg/cfg.go
git add internal/buildcfg/cfg.go
rm internal/buildcfg/cfg_test.go
git add internal/buildcfg/cfg_test.go

rm internal/godebug/godebug_test.go
git add internal/godebug/godebug_test.go

rm internal/godebugs/table.go
git add internal/godebugs/table.go
rmdir internal/godebugs

rm internal/race/doc.go
git add internal/race/doc.go
rm internal/race/norace.go
git add internal/race/norace.go
rm internal/race/race.go
git add internal/race/race.go
rmdir internal/race

rm internal/go/gcimporter/iimport.go
git add internal/go/gcimporter/iimport.go
rm internal/go/typeparams/typeparams.go
git add internal/go/typeparams/typeparams.go
rmdir internal/go/typeparams

LANG=C git status | grep "deleted by us" && exit 1
LANG=C git status | grep "deleted by them" && exit 1

go run $SCRIPT_DIR/fixImports.go upstream-$BRANCH
