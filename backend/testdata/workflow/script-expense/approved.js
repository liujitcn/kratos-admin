// 只输出动作意图，宿主负责校验和事务副作用。
({actions: [{type: "expense.approve", payload: {amount: ctx.variables.total}}]})
