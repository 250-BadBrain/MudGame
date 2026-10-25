// 全局键盘导航：方向键/小键盘数字映射为移动指令。
const keyDirectionMap = {
    ArrowUp: 'north',
    ArrowDown: 'south',
    ArrowLeft: 'west',
    ArrowRight: 'east',
    Numpad8: 'north',
    Numpad2: 'south',
    Numpad4: 'west',
    Numpad6: 'east',
    Numpad3: 'southeast',
    Numpad9: 'northeast',
    Numpad1: 'southwest',
    Numpad7: 'northwest'
}

const isTypingElement = (el) => {
    if (!el || !(el instanceof HTMLElement)) return false
    const tag = el.tagName
    return el.isContentEditable || tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT'
}

export function useKeyNavigation({ onNavigate }) {
    const handleGlobalKeydown = (event) => {
        if (event.defaultPrevented || event.altKey || event.ctrlKey || event.metaKey) return
        if (isTypingElement(event.target)) return

        const direction = keyDirectionMap[event.code]
        if (!direction) return

        event.preventDefault()
        if (document.activeElement instanceof HTMLElement && !isTypingElement(document.activeElement)) {
            document.activeElement.blur()
        }
        window.getSelection()?.removeAllRanges()
        onNavigate(direction)
    }

    return { handleGlobalKeydown }
}
