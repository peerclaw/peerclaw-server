import { useNavigate } from "react-router-dom"
import { useTranslation } from "react-i18next"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { User, Shield, LogOut, ChevronsUpDown } from "lucide-react"
import type { AuthUser } from "@/api/auth"

interface UserMenuProps {
  user: AuthUser
  onLogout: () => void
  showAdminLink?: boolean
}

export function UserMenu({ user, onLogout, showAdminLink }: UserMenuProps) {
  const navigate = useNavigate()
  const { t } = useTranslation()

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button className="flex w-full items-center gap-2.5 rounded-md px-3 py-2 text-sm text-foreground transition-colors hover:bg-accent/50 focus:outline-none">
          <div className="flex size-6 items-center justify-center rounded-full bg-accent text-xs font-medium">
            {(user.display_name || user.email).charAt(0).toUpperCase()}
          </div>
          <span className="flex-1 truncate text-left text-sm font-medium">
            {user.display_name || user.email}
          </span>
          <ChevronsUpDown className="size-4 text-muted-foreground" />
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent side="top" align="start" className="w-56">
        <DropdownMenuLabel className="font-normal">
          <p className="text-sm font-medium truncate">{user.display_name || user.email}</p>
          <p className="text-xs text-muted-foreground truncate">{user.email}</p>
        </DropdownMenuLabel>
        <DropdownMenuSeparator />
        <DropdownMenuItem onClick={() => navigate("/console/profile")}>
          <User />
          {t("nav.profile")}
        </DropdownMenuItem>
        {showAdminLink && (
          <DropdownMenuItem onClick={() => navigate("/admin")}>
            <Shield />
            {t("nav.adminPanel")}
          </DropdownMenuItem>
        )}
        <DropdownMenuSeparator />
        <DropdownMenuItem onClick={onLogout}>
          <LogOut />
          {t("nav.signOut")}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
