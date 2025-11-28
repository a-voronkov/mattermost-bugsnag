import React, {useCallback, useEffect, useState} from 'react';
import PluginSettings from 'components/admin_console/plugin_settings';
import Button from 'components/widgets/buttons';

import manifest from '../../manifest';

interface UserMapping {
    mm_user_id: string;
    bugsnag_user_id?: string;
    bugsnag_email?: string;
}

const UserMappingsSettings: React.FC = () => {
    const [mappings, setMappings] = useState<UserMapping[]>([]);
    const [loading, setLoading] = useState(true);
    const [saving, setSaving] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [success, setSuccess] = useState<string | null>(null);

    const fetchData = useCallback(async () => {
        setLoading(true);
        setError(null);
        try {
            const response = await fetch(`/plugins/${manifest.id}/api/v1/user-mappings`);
            if (response.ok) {
                const data = await response.json();
                setMappings(data.mappings || []);
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

    const handleAdd = useCallback(() => {
        setMappings((prev) => [...prev, {mm_user_id: '', bugsnag_email: ''}]);
    }, []);

    const handleRemove = useCallback((index: number) => {
        setMappings((prev) => prev.filter((_, i) => i !== index));
    }, []);

    const handleChange = useCallback((index: number, field: keyof UserMapping, value: string) => {
        setMappings((prev) => prev.map((m, i) => (i === index ? {...m, [field]: value} : m)));
    }, []);

    const handleSave = useCallback(async () => {
        setSaving(true);
        setError(null);
        setSuccess(null);
        try {
            const response = await fetch(`/plugins/${manifest.id}/api/v1/user-mappings`, {
                method: 'POST',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify({mappings: mappings.filter((m) => m.mm_user_id)}),
            });

            if (!response.ok) {
                const data = await response.json().catch(() => ({}));
                throw new Error(data.error || 'Failed to save');
            }

            setSuccess('User mappings saved successfully');
        } catch (err) {
            setError((err as Error).message);
        } finally {
            setSaving(false);
        }
    }, [mappings]);

    if (loading) {
        return (
            <PluginSettings>
                <div className='form-group'>Loading user mappings...</div>
            </PluginSettings>
        );
    }

    return (
        <PluginSettings>
            <h4>User Mapping (Mattermost ↔ Bugsnag)</h4>
            <p className='help-text'>
                Map Mattermost users to Bugsnag users for assignment actions.
            </p>

            <table className='table'>
                <thead>
                    <tr>
                        <th>Mattermost User ID</th>
                        <th>Bugsnag User ID</th>
                        <th>Bugsnag Email</th>
                        <th>Actions</th>
                    </tr>
                </thead>
                <tbody>
                    {mappings.map((mapping, index) => (
                        <tr key={index}>
                            <td>
                                <input
                                    type='text'
                                    className='form-control'
                                    placeholder='MM User ID'
                                    value={mapping.mm_user_id}
                                    onChange={(e) => handleChange(index, 'mm_user_id', e.target.value)}
                                />
                            </td>
                            <td>
                                <input
                                    type='text'
                                    className='form-control'
                                    placeholder='Bugsnag User ID (optional)'
                                    value={mapping.bugsnag_user_id || ''}
                                    onChange={(e) => handleChange(index, 'bugsnag_user_id', e.target.value)}
                                />
                            </td>
                            <td>
                                <input
                                    type='text'
                                    className='form-control'
                                    placeholder='Bugsnag Email'
                                    value={mapping.bugsnag_email || ''}
                                    onChange={(e) => handleChange(index, 'bugsnag_email', e.target.value)}
                                />
                            </td>
                            <td>
                                <Button btnClass='btn btn-danger btn-sm' onClick={() => handleRemove(index)}>
                                    Remove
                                </Button>
                            </td>
                        </tr>
                    ))}
                </tbody>
            </table>

            <div className='form-group'>
                <Button btnClass='btn btn-default' onClick={handleAdd}>
                    + Add Mapping
                </Button>
            </div>

            <div className='form-group' style={{marginTop: 16}}>
                <Button btnClass='btn btn-primary' disabled={saving} onClick={handleSave}>
                    {saving ? 'Saving...' : 'Save Mappings'}
                </Button>
                {error && <div className='alert alert-danger' style={{marginTop: 12}}>{error}</div>}
                {success && <div className='alert alert-success' style={{marginTop: 12}}>{success}</div>}
            </div>
        </PluginSettings>
    );
};

export default UserMappingsSettings;

