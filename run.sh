#!/bin/bash
# 检查文件是否存在
if [ ! -f "./M3u8Download-Linux-x86_64" ]; then
    echo "错误：找不到可执行文件 ./M3u8Download-Linux-x86_64"
    echo "请确保当前目录下存在名为 'aM3u8Download-Linux-x86_64' 的可执行文件"
    exit 1
fi

# 检查文件是否可执行
if [ ! -x "./M3u8Download-Linux-x86_64" ]; then
    echo "错误：文件 ./M3u8Download-Linux-x86_64 不可执行"
    echo "请使用 chmod +x M3u8Download-Linux-x86_64 命令添加执行权限"
    exit 1
fi

echo "正在执行 ./M3u8Download-Linux-x86_64 ..."
echo "----------------------------------------"
# 执行程序
./M3u8Download-Linux-x86_64
# 获取执行结果
EXIT_CODE=$?
echo "----------------------------------------"
echo "程序执行完成，退出代码: $EXIT_CODE"
# 根据退出代码判断执行结果
if [ $EXIT_CODE -eq 0 ]; then
    echo "✓ 程序执行成功"
else
    echo "✗ 程序执行失败，退出代码: $EXIT_CODE"
fi
exit $EXIT_CODE
