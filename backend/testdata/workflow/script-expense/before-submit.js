// 金额必须为正数；返回值使用对象表达式。
({allowed: Number.isFinite(ctx.form.amount) && ctx.form.amount > 0, message: "金额必须大于零"})
