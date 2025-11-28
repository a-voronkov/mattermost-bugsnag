import React from 'react';

declare module 'components/admin_console/plugin_settings' {
    const PluginSettings: React.ComponentType<any>;
    export default PluginSettings;
}

declare module 'components/admin_console/settings/text_setting' {
    interface TextSettingProps {
        id: string;
        value: string;
        label?: React.ReactNode;
        helpText?: React.ReactNode;
        placeholder?: string;
        onChange?: (id: string, value: string) => void;
        disabled?: boolean;
    }
    const TextSetting: React.ComponentType<TextSettingProps>;
    export default TextSetting;
}

declare module 'components/widgets/buttons' {
    interface ButtonProps {
        onClick?: (event: React.MouseEvent<HTMLButtonElement>) => void;
        btnClass?: string;
        disabled?: boolean;
        children?: React.ReactNode;
    }
    export const Button: React.ComponentType<ButtonProps>;
    export default Button;
}
