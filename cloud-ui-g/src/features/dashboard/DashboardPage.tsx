import LaunchReadinessPanel from './LaunchReadinessPanel';
import PublicationPanel from '../publications/PublicationPanel';
import { usePublication } from '../publications/usePublication';

type DashboardPageProps = {
  restaurantId: string;
};

export default function DashboardPage({ restaurantId }: DashboardPageProps) {
  const { publication } = usePublication(restaurantId);
  return (
    <div className="space-y-4">
      <LaunchReadinessPanel restaurantId={restaurantId} hasPublication={Boolean(publication)} />
      <PublicationPanel restaurantId={restaurantId} canPublish={Boolean(restaurantId)} />
    </div>
  );
}
