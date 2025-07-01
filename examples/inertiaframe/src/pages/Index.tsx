import { usePage } from "@inertiajs/react";

export default function IndexPage() {
  const props = usePage<{ user_id: string }>().props;

  return <div>User ID: {props.user_id}</div>;
}
