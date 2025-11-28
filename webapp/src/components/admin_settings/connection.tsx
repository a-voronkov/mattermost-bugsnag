import React, {useCallback, useMemo, useState} from 'react';
import PluginSettings from 'components/admin_console/plugin_settings';
import TextSetting from 'components/admin_console/settings/text_setting';
import Button from 'components/widgets/buttons';

import manifest from '../../manifest';

const defaultSuccess = 'Connection successful. Bugsnag credentials look valid.';
const defaultFailure = 'Connection failed. Please verify the API token and organization ID.';

const ConnectionSettings: React.FC = () => {
    const [apiToken, setApiToken] = useState('');
    const [orgId, setOrgId] = useState('');
    const [testing, setTesting] = useState(false);
    const [result, setResult] = useState<{status: 'success' | 'error'; message: string} | null>(null);

    const requestBody = useMemo(() => ({
        api_token: apiToken.trim() || undefined,
        organization_id: orgId.trim() || undefined,
    }), [apiToken, orgId]);

    const onTest = useCallback(async () => {
        setTesting(true);
        setResult(null);
        try {
            const response = await fetch(`/plugins/${manifest.id}/api/v1/test`, {
                method: 'POST',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify(requestBody),
            });

            const payload = await response.json().catch(() => ({}));
            if (response.ok) {
                setResult({
                    status: 'success',
                    message: payload.message || defaultSuccess,
                });
                return;
            }

            setResult({
                status: 'error',
                message: payload.message || defaultFailure,
            });
        } catch (error) {
            setResult({
                status: 'error',
                message: (error as Error).message,
            });
        } finally {
            setTesting(false);
        }
    }, [requestBody]);

    return (
        <PluginSettings>
            <TextSetting
                id='bugsnag-api-token'
                value={apiToken}
                label='API Token'
                placeholder='Personal API token'
                helpText='Personal Bugsnag API token used to fetch organizations and projects.'
                onChange={(_, value) => setApiToken(value)}
                disabled={testing}
            />
            <TextSetting
                id='bugsnag-org-id'
                value={orgId}
                label='Organization ID (optional)'
                placeholder='org_12345'
                helpText='Limit API requests to a specific Bugsnag organization.'
                onChange={(_, value) => setOrgId(value)}
                disabled={testing}
            />
            <div className='form-group'>
                <Button
                    btnClass='btn btn-primary'
                    disabled={testing}
                    onClick={onTest}
                >
                    {testing ? 'Testing…' : 'Test connection'}
                </Button>
                {result &&
                <div className={`alert ${result.status === 'success' ? 'alert-success' : 'alert-danger'}`} style={{marginTop: 12}}>
                    {result.message}
                </div>}
            </div>
        </PluginSettings>
    );
};

export default ConnectionSettings;
