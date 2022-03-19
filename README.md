# 双鸭山大学自动化健康申报
> “时间就像海绵里的水，只要愿挤，总还是有的。”
>
> -- <cite>鲁迅</cite>

## 最简客户端
[https://alexzm.xyz:9002](https://alexzm.xyz:9002)

可以的话，使用自己部署的会好一些，这是一件非常简单的事，把对应操作系统的可执行文件下载下来运行即可（在 [Release 页面](https://github.com/jksbx/jksbx/releases)里，十几MB），可以不需要进行配置，程序内置了默认配置。

## 项目外依赖
Chrome浏览器。

## 用法
此程序将默认监听`:8080`端口，通过REST API暴露服务，共有如下API，所有API都是接收POST请求，所有API的请求体里都需要有`username`和`password`字段，分别表示自己的NetID和密码。注意请求体使用经典的`x-www-form-urlencoded`格式，而不是时下流行的`json`。

- `POST /api/submit` 用来发起“尝试提交一次健康申报表”的申请，该申请将会被加到申请队列中排队，过一会应该就可以在微信上收到申报成功提示。
- `POST /api/adduser` 用来将NetID和密码存进数据库里，未来每天早上都会自动申报。
- `POST /api/deleteuser` 用来将NetID和密码从数据库里删除，以后就不会自动申报了。

可以使用上文所述的最简客户端进行一些实验。

## 详细文档
更详细的技术细节以及项目部署文档放在 [`/doc`](/doc) 目录下。

## 其他
目前仅在 64 位 Linux 和 64 位 Windows 上进行过测试（并且能顺利运行），两类 CPU 架构的 MacOS 都没测试过。
