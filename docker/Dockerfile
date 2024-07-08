# 指定镜像目标平台与镜像名 alpine表示基础镜像 第一层镜像
FROM --platform=$TARGETPLATFORM alpine
# 添加元数据到镜像
LABEL maintainer="Jrohy <euvkzx@gmail.com>"
# 编译时变量
ARG TARGETARCH
# 执行编译命令，使用 alpine 的 apk 包管理器安装tzdata libc6-compat gcompat基础镜像不缓存 第二层镜像
RUN apk add --no-cache tzdata libc6-compat gcompat
# 将应用相关文件从构建上下文复制到了当前镜像中，并且新建一个镜像层来存储，第三层镜像
COPY result/webssh_linux_$TARGETARCH /webssh
# 镜像入口点程序，通过镜像元数据的形式保存，不会新增镜像层
ENTRYPOINT ["/webssh"]
