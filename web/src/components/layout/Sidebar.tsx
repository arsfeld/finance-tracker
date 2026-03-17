import { NavLink } from "react-router";

const links = [
  { to: "/", label: "Dashboard" },
  { to: "/transactions", label: "Transactions" },
  { to: "/analysis", label: "Analysis" },
  { to: "/chat", label: "Chat" },
  { to: "/settings", label: "Settings" },
];

export function Sidebar() {
  return (
    <aside className="w-56 border-r border-border bg-card p-4 flex flex-col gap-1">
      <div className="font-semibold text-lg mb-4 px-2">Finance Tracker</div>
      <nav className="flex flex-col gap-0.5">
        {links.map((link) => (
          <NavLink
            key={link.to}
            to={link.to}
            end={link.to === "/"}
            className={({ isActive }) =>
              `px-3 py-2 rounded-md text-sm transition-colors ${
                isActive
                  ? "bg-primary text-primary-foreground font-medium"
                  : "text-muted-foreground hover:bg-accent hover:text-accent-foreground"
              }`
            }
          >
            {link.label}
          </NavLink>
        ))}
      </nav>
    </aside>
  );
}
