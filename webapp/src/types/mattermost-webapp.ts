import React from 'react';

export type PluginComponent = React.ComponentType<any>;

export interface PluginRegistry {
    // Register a custom admin console plugin view for this plugin.
    registerAdminConsolePlugin: (pluginId: string, component: PluginComponent) => void;
}

declare global {
    interface Window {
        registerPlugin: (id: string, plugin: any) => void;
    }
}
