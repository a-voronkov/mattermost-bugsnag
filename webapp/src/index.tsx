import ConnectionSettings from './components/admin_settings/connection';
import manifest from './manifest';
import {PluginRegistry} from './types/mattermost-webapp';

class Plugin {
    public initialize(registry: PluginRegistry) {
        registry.registerAdminConsolePlugin(manifest.id, ConnectionSettings);
    }
}

window.registerPlugin(manifest.id, new Plugin());
