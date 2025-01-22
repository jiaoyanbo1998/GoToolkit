1.基于build模式和Gin框架的JWT中间件，支持token的生成、校验、刷新和获取token中存储的数据。

2.使用哈希算法和Bitmap实现了一个布隆过滤器，用于快速判断元素是否存在。

3.使用Lua脚本实现的滑动窗口限流，防止服务器被大量请求击垮。

4.使用Lua脚本限制用户发送短信的频率和总量，防止非法用户恶意消耗短信资源。

5.使用Lua脚本限制用户验证短信的次数，防止非法用户暴力破解。

6.使用适配器模式和装饰器模式，封装了Zap框架，实现Zap和应用程序解耦，并允许用户自定义配置。

7.封装了Sarama的ConsumerGroupHandler，并提供了kafka消费者批量异步消费的实现。

8.HTTP请求监控中间件

    基于Gin框架的统计HTTP请求响应时间的中间件。

    基于Gin框架的统计当前正在执行的HTTP请求数量的中间件。

    基于Gin框架的统一监控错误码的中间件。

9.通过Redis的Hook函数，监控缓存命中率。

10.通过GORM的回调机制，监控数据库操作的性能。

11.封装了kafka-go客户端，提供了生产者和消费者的实现。

12.封装了MinIO客户端，提供单个文件上，分片上传，删除文件，检测文件是否存在等功能。

13.gRPC服务注册和发现，带有续租机制。

