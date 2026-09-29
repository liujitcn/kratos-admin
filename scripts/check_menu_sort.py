#!/usr/bin/env python3
"""检查菜单种子数据排序是否与编号位置规则一致。

规则：sort 取本级对应位置的两位数字——一级取 1-2 位、二级取 3-4 位、
三级取 5-6 位、四级取 7-8 位；该数字对为 00 时向后回退取下一对
（如按钮 90010001 三级对 5-6=00，回退取 7-8=01；演示页 95000100
二级对 3-4=00，回退取 5-6=01）。

层级：优先沿 parent 链推导；父链不在种子内（如项目只带增量种子）时，
按 id 的非零数字对个数推断层级。

支持两种种子语句，可混合出现在目录内多个 default_data*.up.sql 中：
- INSERT IGNORE INTO `base_menu` ...（新增菜单）；
- UPDATE `base_menu` SET ..., `sort` = N, ... WHERE `id` = X（覆盖更新存量或
  本项目菜单，按出现顺序合并到同一行）。

用法：
    python3 scripts/check_menu_sort.py           # 只读检查，不一致时退出码 1
    python3 scripts/check_menu_sort.py --fix     # 回写种子 SQL（只改 sort）
    python3 scripts/check_menu_sort.py --sql     # 打印同步已初始化库的 UPDATE 语句
"""
import argparse
import json
import re
import sys
from pathlib import Path

DEFAULT_SEED_DIR = Path(__file__).resolve().parent.parent / \
    "backend/migration/assets/v0.0.1/mysql"

INSERT_ROW = re.compile(
    r"^INSERT IGNORE INTO `base_menu` \([^)]*\) VALUES "
    r"\((\d+), (\d+), (\d+), '(.*?)', '(.*?)', '(.*?)', '(.*?)', '(.*?)', '(.*?)', (\d+), (\d+), "
)
UPDATE_ROW = re.compile(r"^UPDATE `base_menu` SET (.*?) WHERE `id` = (\d+)")
SORT_TAIL = re.compile(r"(`sort` = )(\d+)")

# UPDATE SET 子句中可识别的字段（sort 单独处理，用于定位修复位置）。
UPDATE_FIELDS = {
    "parent_id": "parent",
    "type": "mtype",
    "name": "name",
    "meta": "meta",
}


def id_pairs(menu_id):
    return [(menu_id // 10 ** (8 - 2 * i)) % 100 for i in range(1, 5)]


def infer_level(menu_id):
    """父链不可用时，按 id 的非零数字对个数推断层级。"""
    return max(1, sum(1 for p in id_pairs(menu_id) if p != 0))


def expected_sort(menu_id, level):
    idx = min(max(level, 1), 4)
    while idx < 4:
        pair = id_pairs(menu_id)[idx - 1]
        if pair != 0:
            return pair
        idx += 1
    return id_pairs(menu_id)[idx - 1]


def level_of(menu_id, rows):
    depth, cur = 1, menu_id
    while depth < 6:
        row = rows.get(cur)
        if row is None:
            return depth - 1 + infer_level(cur)
        if row["parent"] == 0:
            return depth
        cur = row["parent"]
        depth += 1
    return infer_level(menu_id)


def title_of(row):
    try:
        return json.loads(row["meta"].replace("\\", "")).get("title") or row["name"]
    except Exception:
        return row["name"]


def parse_update_line(line, lineno, path, rows):
    """解析 UPDATE 语句：已有行则合并字段，存量菜单则建虚拟行。
    WHERE `id` IN (...) 之类批量语句不携带单条 id，直接跳过。"""
    m = UPDATE_ROW.match(line)
    if not m:
        return None
    set_clause, menu_id = m.group(1), int(m.group(2))
    fields = {}
    for column, key in UPDATE_FIELDS.items():
        fm = re.search(r"`%s` = (\d+)" % column, set_clause) if key in ("parent", "mtype") \
            else re.search(r"`%s` = '([^']*)'" % column, set_clause)
        if fm:
            fields[key] = int(fm.group(1)) if key in ("parent", "mtype") else fm.group(1)
    sort_ref = None
    sm = re.search(r"`sort` = (\d+)", set_clause)
    if sm:
        fields["sort"] = int(sm.group(1))
        sort_ref = (path, lineno)
    if menu_id in rows:
        rows[menu_id].update(fields)
        if sort_ref:
            rows[menu_id]["sort_ref"] = sort_ref
        return None
    if "sort" not in fields:
        return None  # 未设置 sort 的存量更新，无从校验
    row = dict(kind="update", id=menu_id, parent=0, mtype=0, name="", meta="",
               sort=0, file=path, lineno=lineno, sort_ref=sort_ref)
    row.update(fields)
    return row


def parse_seed(path):
    """path 可以是目录（解析其中 default_data*.up.sql）或单个文件。"""
    if path.is_dir():
        files = sorted(path.glob("default_data*.up.sql"))
        if not files:
            raise SystemExit(f"目录中没有 default_data*.up.sql: {path}")
    else:
        files = [path]
    rows = {}
    for path in files:
        for lineno, line in enumerate(path.read_text(encoding="utf-8").splitlines(keepends=True), 1):
            if "`base_menu`" not in line:
                continue
            if line.startswith("INSERT IGNORE INTO `base_menu`"):
                m = INSERT_ROW.match(line)
                if not m:
                    raise SystemExit(f"{path.name}:{lineno} 无法解析: {line[:80]}")
                g = m.groups()
                row = dict(kind="insert", id=int(g[0]), parent=int(g[1]), mtype=int(g[2]),
                           name=g[4].strip(), meta=g[7], sort=int(g[9]),
                           file=path, lineno=lineno, sort_ref=(path, lineno))
                if row["id"] in rows:
                    # 与 INSERT IGNORE 执行语义一致：先插入者生效，重复 INSERT 跳过。
                    continue
                rows[row["id"]] = row
            elif line.startswith("UPDATE `base_menu` SET"):
                parse_update_line(line, lineno, path, rows)
    for row in rows.values():
        row["level"] = level_of(row["id"], rows)
        row["expect"] = expected_sort(row["id"], row["level"])
    return rows


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--path", type=Path, default=DEFAULT_SEED_DIR,
                    help="种子目录（default_data*.up.sql）或单个 SQL 文件")
    ap.add_argument("--fix", action="store_true", help="把不一致的 sort 回写到种子 SQL")
    ap.add_argument("--sql", action="store_true", help="打印同步数据库的 UPDATE 语句")
    args = ap.parse_args()

    rows = parse_seed(args.path)
    fixes = sorted(
        ((mid, r) for mid, r in rows.items() if r["sort"] != r["expect"]),
        key=lambda x: x[0],
    )

    if args.sql:
        for _, r in fixes:
            print(f"UPDATE base_menu SET sort={r['expect']} WHERE id={r['id']};")
        return 1 if fixes else 0

    tname = {1: "目录", 2: "菜单", 3: "按钮"}
    label = args.path.name if args.path.is_file() else args.path.parent.name + "/"
    print(f"{label} 共 {len(rows)} 条菜单；排序与编号规则不一致 {len(fixes)} 条：")
    if fixes:
        print(f"\n{'ID':<10} {'层级':<4} {'类型':<4} {'名称':<16} {'现sort':>5} -> {'应sort':>4}  修复位置")
        for mid, r in fixes:
            kind = tname.get(r["mtype"], "存量")
            ref = r["sort_ref"]
            print(f"{mid:<10} L{r['level']:<3} {kind:<4} {title_of(r):<16} {r['sort']:>5} -> {r['expect']:>4}  {ref[0].name}:{ref[1]}")

    if args.fix and fixes:
        by_file = {}
        for _, r in fixes:
            by_file.setdefault(r["sort_ref"][0], []).append(r)
        for path, items in by_file.items():
            lines = path.read_text(encoding="utf-8").splitlines(keepends=True)
            for r in sorted(items, key=lambda x: -x["sort_ref"][1]):
                pos = r["sort_ref"][1] - 1
                line = lines[pos]
                m = INSERT_ROW.match(line)
                if m and int(m.group(1)) == r["id"]:
                    lines[pos] = line[:m.start(10)] + str(r["expect"]) + line[m.end(10):]
                else:
                    m = SORT_TAIL.search(line)
                    assert m, f"{path.name}:{pos + 1} 找不到 sort 字段"
                    lines[pos] = line[:m.start(2)] + str(r["expect"]) + line[m.end(2):]
            path.write_text("".join(lines), encoding="utf-8")
        # 写后自检：只允许 sort 变化，其他任何字段被改动都视为失败。
        after = parse_seed(args.path)
        for mid, row in rows.items():
            keys = ("parent", "mtype", "name", "meta")
            before = tuple(row[k] for k in keys)
            new = tuple(after[mid][k] for k in keys)
            assert before == new, f"菜单 {mid} 出现非 sort 字段变动，已中止"
        print(f"\n已回写 {len(fixes)} 处（仅 sort 字段，其余字段逐条校验未变）。")
        return 0  # fix 成功即视为通过
    return 1 if fixes else 0


if __name__ == "__main__":
    sys.exit(main())
