# @local/kratos-taro-app

`apps/taro-app` is the private Taro React host for `__PROJECT_NAME__`. It assembles modules and provides H5 and WeChat Mini Program build entry points; reusable business implementations belong in modules.

Development output is under `apps/taro-app/dist/dev/<platform>` and production output under `apps/taro-app/dist/build/<platform>` (`h5` or `mp-weixin`). For a Kratos-integrated project, the H5 production build is written to `backend/web/taro-app`. WeChat Developer Tools should use `dist/dev/mp-weixin`; import `dist/build/mp-weixin` for release.

The fixed startup page is in `src/pages/bootstrap`. The core runner temporarily generates wrappers, page configuration, and static assets for other module pages during builds, then restores the host directory.

To access H5 over a LAN IP, use the shared certificate under the repository's `certs` directory and set `VITE_APP_HTTPS=true` in `.env.development-h5.local`. If the backend uses HTTPS, also set `VITE_APP_API_URL=https://localhost:7001`.
