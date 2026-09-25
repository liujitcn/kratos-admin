# @local/__MODULE_NAME__

`@local/__MODULE_NAME__` 是 `__PROJECT_NAME__` workspace 内的本地业务模块，通过 `@liujitcn/kratos-uni-app-core` 的公开接口接入宿主。

## 目录结构

```text
packages/modules/__MODULE_NAME__
├── src
│   ├── views               # 模块页面；页面私有组件放在就近 components
│   ├── index.d.mts         # 模块入口类型
│   └── index.mjs           # 页面、视图键和图标注册入口
├── package.json            # 模块名称、依赖和 exports
└── README.md               # 模块职责与维护说明
```

## 开发约束

- 页面放在 `src/views`，并在 `src/index.mjs` 的模块定义中登记页面和稳定视图键。
- 请求、认证、导航和状态能力只通过 `@liujitcn/kratos-uni-app-core` 的公开 exports 使用。
- 页面替换使用稳定视图键，接口不能直接下发任意组件路径。
- 修改模块后从 workspace 根目录运行对应 H5 或微信小程序命令验证装配结果。
