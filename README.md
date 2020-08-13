# near-adapter

## 项目依赖库

- [openwallet](https://github.com/blocktree/openwallet.git)

## 如何测试

openwtester包下的测试用例已经集成了openwallet钱包体系，创建conf目录，新建NEAR.ini文件，编辑如下内容：

```ini
;wallet api url
ServerAPI = "https://localhost:8080"
; ChainID
ChainID = ""
; cache block number
cacheBlockNum = 30
```

## 相关链接

使用类似于bts的账户模型，一个账号可以绑定多个地址，使用账号进行转账

### 区块浏览器

https://explorer.near.org/

### api 文档

https://docs.near.org/docs/interaction/rpc

### 节点搭建

https://docs.near.org/docs/validator/staking

### 项目介绍文档

https://docs.near.org/docs/quick-start