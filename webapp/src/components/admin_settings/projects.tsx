import React, {useCallback, useEffect, useState} from 'react';
import PluginSettings from 'components/admin_console/plugin_settings';
import Button from 'components/widgets/buttons';

import manifest from '../../manifest';

interface Project {
    id: string;
    name: string;
}

interface ChannelRule {
    channel_id: string;
    environments?: string[];
    severities?: string[];
    events?: string[];
}

type ProjectChannelMappings = Record<string, ChannelRule[]>;

const ProjectsSettings: React.FC = () => {
    const [projects, setProjects] = useState<Project[]>([]);
    const [mappings, setMappings] = useState<ProjectChannelMappings>({});
    const [loading, setLoading] = useState(true);
    const [saving, setSaving] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [success, setSuccess] = useState<string | null>(null);

    const fetchData = useCallback(async () => {
        setLoading(true);
        setError(null);
        try {
            const [projectsRes, rulesRes] = await Promise.all([
                fetch(`/plugins/${manifest.id}/api/v1/projects`),
                fetch(`/plugins/${manifest.id}/api/v1/channel-rules`),
            ]);

            if (projectsRes.ok) {
                const data = await projectsRes.json();
                setProjects(data.projects || []);
            }

            if (rulesRes.ok) {
                const data = await rulesRes.json();
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

    const handleChannelChange = useCallback((projectId: string, channelId: string) => {
        setMappings((prev) => ({
            ...prev,
            [projectId]: [{channel_id: channelId}],
        }));
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

            setSuccess('Settings saved successfully');
        } catch (err) {
            setError((err as Error).message);
        } finally {
            setSaving(false);
        }
    }, [mappings]);

    if (loading) {
        return (
            <PluginSettings>
                <div className='form-group'>Loading projects...</div>
            </PluginSettings>
        );
    }

    return (
        <PluginSettings>
            <h4>Project → Channel Mapping</h4>
            <p className='help-text'>
                Configure which Mattermost channel receives notifications for each Bugsnag project.
            </p>

            {projects.length === 0 ? (
                <div className='alert alert-warning'>
                    No projects found. Please configure your Bugsnag API token first.
                </div>
            ) : (
                <table className='table'>
                    <thead>
                        <tr>
                            <th>Project</th>
                            <th>Channel ID</th>
                        </tr>
                    </thead>
                    <tbody>
                        {projects.map((project) => (
                            <tr key={project.id}>
                                <td>{project.name}</td>
                                <td>
                                    <input
                                        type='text'
                                        className='form-control'
                                        placeholder='Channel ID'
                                        value={mappings[project.id]?.[0]?.channel_id || ''}
                                        onChange={(e) => handleChannelChange(project.id, e.target.value)}
                                    />
                                </td>
                            </tr>
                        ))}
                    </tbody>
                </table>
            )}

            <div className='form-group' style={{marginTop: 16}}>
                <Button btnClass='btn btn-primary' disabled={saving} onClick={handleSave}>
                    {saving ? 'Saving...' : 'Save Settings'}
                </Button>
                {error && <div className='alert alert-danger' style={{marginTop: 12}}>{error}</div>}
                {success && <div className='alert alert-success' style={{marginTop: 12}}>{success}</div>}
            </div>
        </PluginSettings>
    );
};

export default ProjectsSettings;

