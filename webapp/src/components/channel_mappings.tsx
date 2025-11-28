import React, {useState, useEffect, useCallback} from 'react';

interface ChannelRule {
    id: string;
    project_id: string;
    project_name: string;
    channel_id: string;
    channel_name: string;
}

interface Channel { id: string; display_name: string; name: string; }
interface Project { id: string; name: string; }
interface Props { id: string; value: string; onChange: (id: string, value: string) => void; setSaveNeeded: () => void; }

const styles: {[key: string]: React.CSSProperties} = {
    container: {padding: '10px 0'},
    error: {color: '#d24b4e', marginBottom: '10px'},
    addForm: {display: 'flex', gap: '10px', marginBottom: '15px', flexWrap: 'wrap'},
    select: {padding: '8px 12px', borderRadius: '4px', border: '1px solid #ccc', minWidth: '200px'},
    button: {padding: '8px 16px', borderRadius: '4px', border: 'none', backgroundColor: '#166de0', color: '#fff', cursor: 'pointer'},
    table: {width: '100%', borderCollapse: 'collapse'},
    th: {textAlign: 'left', padding: '10px', borderBottom: '2px solid #ddd', backgroundColor: '#f5f5f5'},
    td: {padding: '10px', borderBottom: '1px solid #eee'},
    deleteBtn: {padding: '4px 8px', borderRadius: '4px', border: '1px solid #d24b4e', backgroundColor: '#fff', color: '#d24b4e', cursor: 'pointer'},
    empty: {textAlign: 'center', color: '#888', padding: '20px'},
};

const ChannelMappings: React.FC<Props> = ({setSaveNeeded}) => {
    const [rules, setRules] = useState<ChannelRule[]>([]);
    const [projects, setProjects] = useState<Project[]>([]);
    const [channels, setChannels] = useState<Channel[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [newRule, setNewRule] = useState({projectId: '', channelId: ''});

    const fetchData = useCallback(async () => {
        try {
            setLoading(true);
            const [rulesRes, projectsRes, channelsRes] = await Promise.all([
                fetch('/plugins/com.mattermost.bugsnag/api/v1/channel-rules'),
                fetch('/plugins/com.mattermost.bugsnag/api/v1/projects'),
                fetch('/api/v4/channels'),
            ]);
            if (rulesRes.ok) { const d = await rulesRes.json(); setRules(d.rules || []); }
            if (projectsRes.ok) { const d = await projectsRes.json(); setProjects(d.projects || []); }
            if (channelsRes.ok) { setChannels(await channelsRes.json() || []); }
            setError(null);
        } catch (e) { setError('Failed to load data'); }
        finally { setLoading(false); }
    }, []);

    useEffect(() => { fetchData(); }, [fetchData]);

    const saveRules = async (updated: ChannelRule[]) => {
        const res = await fetch('/plugins/com.mattermost.bugsnag/api/v1/channel-rules', {
            method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({rules: updated}),
        });
        if (res.ok) { setRules(updated); setSaveNeeded(); }
    };

    const addRule = async () => {
        if (!newRule.projectId || !newRule.channelId) return;
        const proj = projects.find((p) => p.id === newRule.projectId);
        const chan = channels.find((c) => c.id === newRule.channelId);
        await saveRules([...rules, {
            id: `${Date.now()}`, project_id: newRule.projectId, project_name: proj?.name || newRule.projectId,
            channel_id: newRule.channelId, channel_name: chan?.display_name || chan?.name || newRule.channelId,
        }]);
        setNewRule({projectId: '', channelId: ''});
    };

    if (loading) return <div style={styles.container}>Loading...</div>;

    return (
        <div style={styles.container}>
            {error && <div style={styles.error}>{error}</div>}
            <div style={styles.addForm}>
                <select style={styles.select} value={newRule.projectId} onChange={(e) => setNewRule({...newRule, projectId: e.target.value})}>
                    <option value="">Select Bugsnag Project</option>
                    {projects.map((p) => <option key={p.id} value={p.id}>{p.name}</option>)}
                </select>
                <select style={styles.select} value={newRule.channelId} onChange={(e) => setNewRule({...newRule, channelId: e.target.value})}>
                    <option value="">Select Channel</option>
                    {channels.map((c) => <option key={c.id} value={c.id}>{c.display_name || c.name}</option>)}
                </select>
                <button style={styles.button} onClick={addRule}>Add Mapping</button>
            </div>
            <table style={styles.table}>
                <thead><tr><th style={styles.th}>Bugsnag Project</th><th style={styles.th}>Mattermost Channel</th><th style={styles.th}>Actions</th></tr></thead>
                <tbody>
                    {rules.map((r) => (
                        <tr key={r.id}>
                            <td style={styles.td}>{r.project_name}</td>
                            <td style={styles.td}>{r.channel_name}</td>
                            <td style={styles.td}><button style={styles.deleteBtn} onClick={() => saveRules(rules.filter((x) => x.id !== r.id))}>Remove</button></td>
                        </tr>
                    ))}
                    {rules.length === 0 && <tr><td colSpan={3} style={styles.empty}>No mappings configured</td></tr>}
                </tbody>
            </table>
        </div>
    );
};

export default ChannelMappings;