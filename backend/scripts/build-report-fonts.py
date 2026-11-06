"""为报告镜像生成完整的静态 TrueType 中文字体。"""

import base64
import hashlib
import io
import json
from pathlib import Path
import urllib.request

from fontTools.ttLib import TTFont
from fontTools.varLib.instancer import instantiateVariableFont

# 固定官方字体 blob 和校验值，避免构建结果跟随上游分支变化。
source = "https://api.github.com/repos/notofonts/noto-cjk/git/blobs/5371a543be5fc670c7cdee9760c03554ee3e9b8e"
request = urllib.request.Request(source, headers={"User-Agent": "kratos-admin-report-build"})
with urllib.request.urlopen(request, timeout=120) as response:
    content = base64.b64decode(json.load(response)["content"])
if hashlib.sha256(content).hexdigest() != "d68bafcb48a2707749396aa12bbbd833cb70401f3a9a689fd2902c7e0d295964":
    raise ValueError("中文字体校验失败")

output = Path("/fonts")
output.mkdir()
font = TTFont(io.BytesIO(content))
for weight, name in [(400, "Regular"), (700, "Bold")]:
    static = instantiateVariableFont(font, {"wght": weight}, inplace=False, updateFontNames=True)
    if "glyf" not in static or "fvar" in static:
        raise ValueError("报告字体必须为静态 TrueType")
    static.save(output / f"NotoSansSC-{name}.ttf")
