import manifest from './manifest';
import ChannelMappings from './components/channel_mappings';
import UserMappings from './components/user_mappings';

// Plugin registry interface for Mattermost plugins
interface PluginRegistry {
    registerAdminConsoleCustomSetting?: (key: string, component: React.ComponentType<any>) => void;
}

class Plugin {
    public initialize(registry: PluginRegistry): void {
        // eslint-disable-next-line no-console
        console.log(`[${manifest.id}] Initializing plugin, registry methods:`, Object.keys(registry));

        // Register custom admin console settings for mappings
        if (registry.registerAdminConsoleCustomSetting) {
            registry.registerAdminConsoleCustomSetting('ChannelMappings', ChannelMappings);
            registry.registerAdminConsoleCustomSetting('UserMappings', UserMappings);
            // eslint-disable-next-line no-console
            console.log(`[${manifest.id}] Registered custom admin settings: ChannelMappings, UserMappings`);
        } else {
            // eslint-disable-next-line no-console
            console.warn(`[${manifest.id}] registerAdminConsoleCustomSetting not available in registry`);
        }
    }
}

declare global {
    interface Window {
        registerPlugin(pluginId: string, plugin: Plugin): void;
    }
}

window.registerPlugin(manifest.id, new Plugin());
