# @local/kratos-taro-app

`apps/taro-app` 是 `__PROJECT_NAME__` 的私有 Taro React 宿主，负责装配模块并提供 H5、微信小程序构建入口，不承载可复用业务实现。

开发产物位于 apps/taro-app/dist/dev/<平台>，生产产物位于 apps/taro-app/dist/build/<平台>（平台为 h5 或 mp-weixin）；Kratos 配套项目的 H5 生产产物输出到 backend/web/taro-app。微信开发者工具默认使用 dist/dev/mp-weixin，发布时导入 dist/build/mp-weixin。

固定启动页位于 `src/pages/bootstrap`。其他模块页面由 core runner 在构建期间临时生成包装器、页面配置与静态资源，构建结束后会自动恢复宿主目录。

通过局域网 IP 访问 H5 时，可复用仓库根 certs 下的共享证书，并在 .env.development-h5.local 中设置 VITE_APP_HTTPS=true；后端使用 HTTPS 时同步设置 VITE_APP_API_URL=https://localhost:7001。
