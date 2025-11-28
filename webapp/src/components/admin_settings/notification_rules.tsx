import React, {useCallback, useEffect, useState} from 'react';
import PluginSettings from 'components/admin_console/plugin_settings';
import Button from 'components/widgets/buttons';

import manifest from '../../manifest';

interface ChannelRule {
    channel_id: string;
    environments?: string[];
    severities?: string[];
    events?: string[];
}

type ProjectChannelMappings = Record<string, ChannelRule[]>;

const SEVERITIES = ['error', 'warning', 'info'];
const EVENTS = ['exception', 'error', 'firstException', 'reopened', 'spikeStart', 'spikeEnd'];

const NotificationRulesSettings: React.FC = () => {
    const [mappings, setMappings] = useState<ProjectChannelMappings>({});
    const [loading, setLoading] = useState(true);
    const [saving, setSaving] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [success, setSuccess] = useState<string | null>(null);

    const fetchData = useCallback(async () => {
        setLoading(true);
        setError(null);
        try {
            const response = await fetch(`/plugins/${manifest.id}/api/v1/channel-rules`);
            if (response.ok) {
                const data = await response.json();
                setMappings(data.mappings || {});
            }
        } catch (err) {
            setError((err as Error).message);
        } finally {
            setLoading(false);
        }
    }, []);

    useEffect(() => {
        fetchData();
    }, [fetchData]);

    const handleRuleChange = useCallback((
        projectId: string,
        ruleIndex: number,
        field: keyof ChannelRule,
        value: string | string[],
    ) => {
        setMappings((prev) => {
            const rules = [...(prev[projectId] || [])];
            rules[ruleIndex] = {...rules[ruleIndex], [field]: value};
            return {...prev, [projectId]: rules};
        });
    }, []);

    const handleSave = useCallback(async () => {
        setSaving(true);
        setError(null);
        setSuccess(null);
        try {
            const response = await fetch(`/plugins/${manifest.id}/api/v1/channel-rules`, {
                method: 'POST',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify({mappings}),
            });

            if (!response.ok) {
                const data = await response.json().catch(() => ({}));
                throw new Error(data.error || 'Failed to save');
            }

            setSuccess('Notification rules saved successfully');
        } catch (err) {
            setError((err as Error).message);
        } finally {
            setSaving(false);
        }
    }, [mappings]);

    if (loading) {
        return (
            <PluginSettings>
                <div className='form-group'>Loading notification rules...</div>
            </PluginSettings>
        );
    }

    const projectIds = Object.keys(mappings);

    return (
        <PluginSettings>
            <h4>Notification Rules</h4>
            <p className='help-text'>
                Configure filters for each project-channel mapping. Leave empty to receive all notifications.
            </p>

            {projectIds.length === 0 ? (
                <div className='alert alert-info'>
                    No project mappings configured. Add project→channel mappings first.
                </div>
            ) : (
                projectIds.map((projectId) => (
                    <div key={projectId} className='panel panel-default' style={{marginBottom: 16}}>
                        <div className='panel-heading'>
                            <strong>Project: {projectId}</strong>
                        </div>
                        <div className='panel-body'>
                            {(mappings[projectId] || []).map((rule, ruleIndex) => (
                                <div key={ruleIndex} style={{marginBottom: 12}}>
                                    <div className='form-group'>
                                        <label>Environments (comma-separated)</label>
                                        <input
                                            type='text'
                                            className='form-control'
                                            placeholder='production, staging'
                                            value={(rule.environments || []).join(', ')}
                                            onChange={(e) => handleRuleChange(
                                                projectId,
                                                ruleIndex,
                                                'environments',
                                                e.target.value.split(',').map((s) => s.trim()).filter(Boolean),
                                            )}
                                        />
                                    </div>
                                    <div className='form-group'>
                                        <label>Severities</label>
                                        <div>
                                            {SEVERITIES.map((sev) => (
                                                <label key={sev} style={{marginRight: 12}}>
                                                    <input
                                                        type='checkbox'
                                                        checked={(rule.severities || []).includes(sev)}
                                                        onChange={(e) => {
                                                            const current = rule.severities || [];
                                                            const updated = e.target.checked
                                                                ? [...current, sev]
                                                                : current.filter((s) => s !== sev);
                                                            handleRuleChange(projectId, ruleIndex, 'severities', updated);
                                                        }}
                                                    />
                                                    {' '}{sev}
                                                </label>
                                            ))}
                                        </div>
                                    </div>
                                    <div className='form-group'>
                                        <label>Events</label>
                                        <div>
                                            {EVENTS.map((evt) => (
                                                <label key={evt} style={{marginRight: 12}}>
                                                    <input
                                                        type='checkbox'
                                                        checked={(rule.events || []).includes(evt)}
                                                        onChange={(e) => {
                                                            const current = rule.events || [];
                                                            const updated = e.target.checked
                                                                ? [...current, evt]
                                                                : current.filter((ev) => ev !== evt);
                                                            handleRuleChange(projectId, ruleIndex, 'events', updated);
                                                        }}
                                                    />
                                                    {' '}{evt}
                                                </label>
                                            ))}
                                        </div>
                                    </div>
                                </div>
                            ))}
                        </div>
                    </div>
                ))
            )}

            <div className='form-group' style={{marginTop: 16}}>
                <Button btnClass='btn btn-primary' disabled={saving} onClick={handleSave}>
                    {saving ? 'Saving...' : 'Save Rules'}
                </Button>
                {error && <div className='alert alert-danger' style={{marginTop: 12}}>{error}</div>}
                {success && <div className='alert alert-success' style={{marginTop: 12}}>{success}</div>}
            </div>
        </PluginSettings>
    );
};

export default NotificationRulesSettings;

