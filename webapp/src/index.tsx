import AdminSettings from './components/admin_settings';
import manifest from './manifest';
import {PluginRegistry} from './types/mattermost-webapp';

class Plugin {
    public initialize(registry: PluginRegistry) {
        registry.registerAdminConsolePlugin(manifest.id, AdminSettings);
    }
}

window.registerPlugin(manifest.id, new Plugin());
