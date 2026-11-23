import { TicketDetailClient } from "../../../../components/tickets/TicketDetailClient";

interface Props {
  params: Promise<{ id: string }>;
}

export default async function TicketDetailPage({ params }: Props) {
  const { id } = await params;
  return <TicketDetailClient ticketId={id} />;
}
