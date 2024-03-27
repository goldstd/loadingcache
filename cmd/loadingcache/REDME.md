# 测试

1. 启动 mysql 服务 (cd testdata/; docker-compose up -d)
2. 启动 loadingcache 服务
3. 使用 godbtest 每5秒更新一次数据
4. 启动 berf 压测 ` SAMPLING_RATE=0.00001 berf :8080 -pb` 观察数据更新情况

## godbtest

```sh
$ godbtest
> %connect mysql://root:root@127.0.0.1:3306/gcache;
> %set --think 5s --times 10;
> update gcache set v = '@姓名';
```

## berf

```sh
$ SAMPLING_RATE=0.00001 berf :8080 -pb
Berf  http://127.0.0.1:8080/ using 100 goroutine(s), 12 GoMaxProcs.
"湛硘泫"

"湛硘泫"

"湛硘泫"

"卓纲雐"

"卓纲雐"

"卓纲雐"

"卓纲雐"

```
