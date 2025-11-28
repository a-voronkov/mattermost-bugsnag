import React, {useState, useEffect, useCallback} from 'react';

interface UserMapping {
    mattermost_user_id: string;
    mattermost_username: string;
    bugsnag_user_id: string;
    bugsnag_email: string;
}

interface MMUser { id: string; username: string; email: string; }
interface BugsnagUser { id: string; name: string; email: string; }
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

const UserMappings: React.FC<Props> = ({setSaveNeeded}) => {
    const [mappings, setMappings] = useState<UserMapping[]>([]);
    const [users, setUsers] = useState<MMUser[]>([]);
    const [bugsnagUsers, setBugsnagUsers] = useState<BugsnagUser[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [newMapping, setNewMapping] = useState({mmUserId: '', bugsnagUserId: ''});

    const fetchData = useCallback(async () => {
        try {
            setLoading(true);
            const [mappingsRes, usersRes, bugsnagRes] = await Promise.all([
                fetch('/plugins/com.mattermost.bugsnag/api/v1/user-mappings'),
                fetch('/api/v4/users?per_page=200'),
                fetch('/plugins/com.mattermost.bugsnag/api/v1/collaborators'),
            ]);
            if (mappingsRes.ok) { const d = await mappingsRes.json(); setMappings(d.mappings || []); }
            if (usersRes.ok) { setUsers(await usersRes.json() || []); }
            if (bugsnagRes.ok) { const d = await bugsnagRes.json(); setBugsnagUsers(d.collaborators || []); }
            setError(null);
        } catch (e) { setError('Failed to load data'); }
        finally { setLoading(false); }
    }, []);

    useEffect(() => { fetchData(); }, [fetchData]);

    const saveMappings = async (updated: UserMapping[]) => {
        const res = await fetch('/plugins/com.mattermost.bugsnag/api/v1/user-mappings', {
            method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({mappings: updated}),
        });
        if (res.ok) { setMappings(updated); setSaveNeeded(); }
    };

    const addMapping = async () => {
        if (!newMapping.mmUserId || !newMapping.bugsnagUserId) return;
        const mmUser = users.find((u) => u.id === newMapping.mmUserId);
        const bsUser = bugsnagUsers.find((u) => u.id === newMapping.bugsnagUserId);
        await saveMappings([...mappings, {
            mattermost_user_id: newMapping.mmUserId,
            mattermost_username: mmUser?.username || newMapping.mmUserId,
            bugsnag_user_id: newMapping.bugsnagUserId,
            bugsnag_email: bsUser?.email || '',
        }]);
        setNewMapping({mmUserId: '', bugsnagUserId: ''});
    };

    if (loading) return <div style={styles.container}>Loading...</div>;

    return (
        <div style={styles.container}>
            {error && <div style={styles.error}>{error}</div>}
            <div style={styles.addForm}>
                <select style={styles.select} value={newMapping.mmUserId} onChange={(e) => setNewMapping({...newMapping, mmUserId: e.target.value})}>
                    <option value="">Select Mattermost User</option>
                    {users.map((u) => <option key={u.id} value={u.id}>{u.username} ({u.email})</option>)}
                </select>
                <select style={styles.select} value={newMapping.bugsnagUserId} onChange={(e) => setNewMapping({...newMapping, bugsnagUserId: e.target.value})}>
                    <option value="">Select Bugsnag User</option>
                    {bugsnagUsers.map((u) => <option key={u.id} value={u.id}>{u.name} ({u.email})</option>)}
                </select>
                <button style={styles.button} onClick={addMapping}>Add Mapping</button>
            </div>
            <table style={styles.table}>
                <thead><tr><th style={styles.th}>Mattermost User</th><th style={styles.th}>Bugsnag User</th><th style={styles.th}>Actions</th></tr></thead>
                <tbody>
                    {mappings.map((m, i) => (
                        <tr key={m.mattermost_user_id || i}>
                            <td style={styles.td}>{m.mattermost_username || m.mattermost_user_id}</td>
                            <td style={styles.td}>{m.bugsnag_email || m.bugsnag_user_id}</td>
                            <td style={styles.td}><button style={styles.deleteBtn} onClick={() => saveMappings(mappings.filter((x) => x.mattermost_user_id !== m.mattermost_user_id))}>Remove</button></td>
                        </tr>
                    ))}
                    {mappings.length === 0 && <tr><td colSpan={3} style={styles.empty}>No user mappings configured</td></tr>}
                </tbody>
            </table>
        </div>
    );
};

export default UserMappings;