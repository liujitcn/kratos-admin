#!/usr/bin/env python3
"""保留工作流大 JSON 字段的跨数据库容量，避免 AutoMigrate 缩小列类型。"""
from pathlib import Path
import re

MODELS = Path(__file__).resolve().parents[1] / "internal/data/gen/models"
COLUMNS = {
    "workflow_definition": {"node_config"},
    "workflow_instance": {"node_config", "variables"},
    "workflow_script_run": {"input_json", "output_json"},
    "workflow_task": set(),
}
TIME_POINTER_FIELDS = {
    "workflow_instance": {"EndAt"},
    "workflow_task": {"RemindedAt", "EscalatedAt"},
}

for table, columns in COLUMNS.items():
    path = MODELS / f"{table}.gen.go"
    if not path.exists():
        continue  # 支持 gorm-gen 的单表生成模式。
    source = path.read_text()
    for column in columns:
        pattern = rf"(gorm:\"column:{column};)type:(?:text|mediumtext|string)(?:;size:\d+)?;"
        source, count = re.subn(pattern, r"\1type:string;size:1048576;", source)
        if count != 1:
            raise SystemExit(f"未找到唯一字段声明: {table}.{column}")
    for field in TIME_POINTER_FIELDS.get(table, set()):
        pattern = rf"(\t{field}\s+)time\.Time(\s+`gorm:\")"
        source, count = re.subn(pattern, r"\1*time.Time\2", source)
        if count != 1:
            raise SystemExit(f"未找到唯一可空时间字段: {table}.{field}")
    path.write_text(source)
