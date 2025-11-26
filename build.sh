# build命令
RUN_NAME="achobeta.server.forge"
mkdir -p output/bin output/conf
cp script/* output/
cp -r conf/* output/conf/
chmod +x output/bootstrap.sh

go build -o output/bin/${RUN_NAME} ./cmd  -coverpkg=./...