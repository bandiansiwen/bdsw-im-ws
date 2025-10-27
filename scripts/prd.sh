# 使用环境变量
export ENV=production
export NACOS_SERVER_ADDR=nacos-cluster:8848
./main

# 或者使用 Docker
docker run -e ENV=production -e NACOS_SERVER_ADDR=nacos:8848 your-image