export interface AppConfig {
    video: {
        width: number;
        height: number;
    };
    chat: {
        cooldown_ms: number;
        max_messages: number;
    };
    commands: {
        standard: string[];
        parameterized: {
            [key: string]: { min: number; max: number };
        };
    };
}

export function isValidAppConfig(data: any): data is AppConfig {
    if (!data || typeof data !== 'object') return false;

    // video
    if (!data.video || 
        typeof data.video.width !== 'number' || 
        typeof data.video.height !== 'number') {
        return false;
    }

    // chat
    if (!data.chat || 
        typeof data.chat.cooldown_ms !== 'number' || 
        typeof data.chat.max_messages !== 'number') {
        return false;
    }

    // cmds
    if (!data.commands || 
        !Array.isArray(data.commands.standard) || 
        !data.commands.parameterized || 
        typeof data.commands.parameterized !== 'object') {
        return false;
    }

    // parameterized
    for (const key in data.commands.parameterized) {
        const bounds = data.commands.parameterized[key];
        if (!bounds || typeof bounds.min !== 'number' || typeof bounds.max !== 'number') {
            return false;
        }
    }

    return true;
}