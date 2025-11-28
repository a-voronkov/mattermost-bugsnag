import React, {useState} from 'react';

import ConnectionSettings from './connection';
import ProjectsSettings from './projects';
import UserMappingsSettings from './user_mappings';
import NotificationRulesSettings from './notification_rules';

type TabId = 'connection' | 'projects' | 'users' | 'rules';

interface Tab {
    id: TabId;
    label: string;
    component: React.ComponentType;
}

const TABS: Tab[] = [
    {id: 'connection', label: 'Connection', component: ConnectionSettings},
    {id: 'projects', label: 'Projects & Channels', component: ProjectsSettings},
    {id: 'users', label: 'User Mapping', component: UserMappingsSettings},
    {id: 'rules', label: 'Notification Rules', component: NotificationRulesSettings},
];

const AdminSettings: React.FC = () => {
    const [activeTab, setActiveTab] = useState<TabId>('connection');

    const ActiveComponent = TABS.find((t) => t.id === activeTab)?.component || ConnectionSettings;

    return (
        <div className='bugsnag-admin-settings'>
            <ul className='nav nav-tabs' style={{marginBottom: 20}}>
                {TABS.map((tab) => (
                    <li
                        key={tab.id}
                        className={activeTab === tab.id ? 'active' : ''}
                    >
                        <a
                            href='#'
                            onClick={(e) => {
                                e.preventDefault();
                                setActiveTab(tab.id);
                            }}
                        >
                            {tab.label}
                        </a>
                    </li>
                ))}
            </ul>
            <ActiveComponent />
        </div>
    );
};

export default AdminSettings;

