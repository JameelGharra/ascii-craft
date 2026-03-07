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
    return data
        && data.video && typeof data.video.width === 'number'
        && data.chat && typeof data.chat.cooldown_ms === 'number'
        && data.commands && Array.isArray(data.commands.standard);
}