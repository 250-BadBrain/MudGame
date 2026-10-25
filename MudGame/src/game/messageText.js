// 指令响应的用户文案映射：根据 (type, subtype, flag, error) 生成
// 优先展示的日志文本；返回 null 表示交由通用处理。
export const describeCommandError = (error, fallback) => {
    if (typeof error === 'string' && /[\u3400-\u9fff]/.test(error)) return error
    return fallback
}

export const getMessage = (msg) => {
    const { type, subtype, flag, results } = msg

    if (type === 'command') {
        if (subtype === 'heal') {
            if (flag) return "你开始运功疗伤..."
            const err = results ? results.error : undefined
            if (err === 'combat_state') return "战斗中无法疗伤！"
            if (err === 'dead_state') return "死人是无法疗伤的。"
            if (err === 'learning_state') return "你正在学习，无法分心疗伤。"
            if (err === 'hp_full') return "你气血充沛，不需要疗伤"
            if (err === 'mp_insufficient') return "你的内力不足，无法疗伤。"
            return describeCommandError(err, "你现在无法运功疗伤。")
        }
        if (subtype === 'meditate') {
            if (flag) {
                // 如果有具体消息（如内力增加），则不显示默认的开始消息，避免重复或冗余
                return (results && results.message) ? null : "你盘膝而坐，开始吐纳调息..."
            }
            const err = results ? results.error : 'unknown_error'
            if (err === 'combat_state') return "战斗中无法打坐！"
            if (err === 'dead_state') return "死人是无法打坐的。"
            if (err === 'learning_state') return "你正在学习，无法同时打坐。"
            if (err === 'full_state') return "你现在精力充沛，无需打坐。"
            return describeCommandError(err, "你现在无法静心打坐。")
        }
        if (subtype === 'go') {
            if (flag) {
                if (results.room) return `你来到了${results.room.name}`
                return null
            }
            const err = results ? results.error : undefined
            if (err === 'combat_state') return "你正在战斗中，无法脱身！"
            if (err === 'dead_state') return "你已经死了，动弹不得。"
            if (err === 'learning_state') return "你正在学习，无法随意走动。"
            if (err === 'no_exit_in_that_direction') return "这个方向没有路。"
            return describeCommandError(err, "你现在无法向这个方向移动。")
        }
        if (subtype === 'spar') {
            if (flag) return results?.message || null
            const err = results ? results.error : undefined
            if (err === 'combat_state') return "你正在战斗中。"
            if (err === 'dead_state') return "你已经死了，无法切磋。"
            if (err === 'busy') return "你现在正忙着呢。"
            if (err === 'self_not_full') return "你先调整好自己的状态再跟人家比试吧！"
            if (err === 'target_not_full') return `${results.targetName || '对方'}状态不好，你想趁人之危吗？`
            if (err === 'target_missing') return "目标不能为空"
            if (err === 'self_target') return "和自己比试就不必了。"
            if (err === 'target_not_found') return "找不到目标"
            if (err === 'npc_combat_unsupported') return "当前版本暂不支持与 NPC 切磋。"
            if (err === 'target_not_in_room') return "目标不在当前房间"
            if (err === 'target_is_dead') return "你要比试谁？"
            if (err === 'target_busy') return `${results.targetName || '对方'}现在正忙着。`
            return describeCommandError(err, "眼下并不适合切磋。")
        }
        if (subtype === 'kill') {
            if (flag) return null
            const err = results ? results.error : undefined
            if (err === 'busy_state') return "你现在正忙着呢。"
            if (err === 'dead_state') return "你已经死了，无法攻击。"
            if (err === 'target_missing') return "目标不能为空"
            if (err === 'self_target') return "你总不能对自己下手吧？"
            if (err === 'target_in_combat') return `${results.targetName || '对方'}正在战斗中。`
            if (err === 'target_not_found') return "找不到目标"
            if (err === 'target_not_in_room') return "目标不在当前房间"
            if (err === 'npc_combat_unsupported') return "当前版本暂不支持与 NPC 战斗。"
            if (err === 'target_is_dead') return "你要击杀谁？"
            return describeCommandError(err, "你现在无法向对方出手。")
        }
        if (subtype === 'learn') {
            if (flag) return null
            const err = results ? results.error : undefined
            if (err === 'busy_state') return "你现在正忙着呢。"
            if (err === 'no_potential') return "你的潜能不足，无法继续学习。"
            if (err === 'skill_not_taught') return "这位老师似乎不会这招。"
            if (err === 'npc_not_found') return "找不到这位老师。"
            if (typeof err === 'string' && err.includes('已经不输你师傅了')) return err
            return describeCommandError(err, "你现在无法继续学习这门武功。")
        }
        if (subtype === 'stop') {
            if (flag) return results?.message || null
            if (results?.error === 'escape_failed') return "你想抽身后退，却被对手缠得脱不开身。"
            return describeCommandError(results?.error, "你一时无法停下当前动作。")
        }
        if (subtype === 'enter_dungeon') {
            if (flag) return null
            const err = results?.error
            if (err === 'already_in_dungeon') return "你已经在副本中了。"
            if (err === 'dungeon_not_found') return "没有找到这个副本。"
            if (err === 'dungeon_unavailable') return "副本系统暂不可用。"
            if (err === 'busy_state') return "你现在正忙着呢。"
            if (err === 'dead_state') return "你已经重伤不起，先复活再说。"
            return describeCommandError(err, "你现在无法进入这处秘境。")
        }
        if (subtype === 'leave_dungeon') {
            if (flag) return null
            const err = results?.error
            if (err === 'not_in_dungeon') return "你当前不在副本中。"
            return describeCommandError(err, "你现在无法离开这处秘境。")
        }
    }
    return null
}
