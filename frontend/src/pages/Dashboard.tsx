import { useState } from 'react';
import { CreateServerForm } from '../components/CreateServerForm';
import { ServerList } from '../components/ServerList';

interface Props {
  onManage: (uuid: string) => void;
}

export function Dashboard({ onManage }: Props) {
  const [refreshKey, setRefreshKey] = useState(0);

  return (
    <div className="view active">
      <div className="dash-head">
        <h1>Servers</h1>
        <p>Everything you have access to, across every node.</p>
      </div>
      <CreateServerForm onCreated={() => setRefreshKey((k) => k + 1)} />
      <ServerList key={refreshKey} onManage={onManage} />
    </div>
  );
}
