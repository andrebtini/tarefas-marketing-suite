/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { HelpCircle, MoveLeft } from "lucide-react";
import { MS_URL_GUIA_INTERNO, WEB_BASE_URL } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
// plane internal packages
import { NewTabIcon } from "@plane/propel/icons";
import { Tooltip } from "@plane/propel/tooltip";
import { cn } from "@plane/utils";
// hooks
import { useTheme } from "@/hooks/store";

export const AdminSidebarHelpSection = observer(function AdminSidebarHelpSection() {
  const { isSidebarCollapsed, toggleSidebar } = useTheme();
  const { t } = useTranslation();
  const redirectionLink = encodeURI(WEB_BASE_URL + "/");

  return (
    <div
      className={cn(
        "flex h-14 w-full flex-shrink-0 items-center justify-between gap-1 self-baseline border-t border-subtle bg-surface-1 px-4",
        {
          "h-auto flex-col py-1.5": isSidebarCollapsed,
        }
      )}
    >
      <div className={`flex items-center gap-1 ${isSidebarCollapsed ? "flex-col justify-center" : "w-full"}`}>
        <Tooltip tooltipContent="Redirect to Plane" position="right" className="ml-4" disabled={!isSidebarCollapsed}>
          <a
            href={redirectionLink}
            className={`relative flex items-center gap-1 rounded-sm bg-layer-1 px-2 py-1 text-body-xs-medium whitespace-nowrap text-secondary`}
          >
            <NewTabIcon width={14} height={14} />
            {!isSidebarCollapsed && "Redirect to Plane"}
          </a>
        </Tooltip>
        {/* MS: sem URL de guia, o botao de ajuda nao aparece (MRC-002) */}
        {MS_URL_GUIA_INTERNO ? (
          <Tooltip tooltipContent={t("ms.common.guia_interno")} position={isSidebarCollapsed ? "right" : "top"} className="ml-4">
            <a
              href={MS_URL_GUIA_INTERNO}
              target="_blank"
              rel="noopener noreferrer"
              aria-label={t("ms.common.guia_interno")}
              className={`ml-auto grid place-items-center rounded-md p-1.5 text-secondary outline-none hover:bg-layer-1-hover hover:text-primary ${
                isSidebarCollapsed ? "w-full" : ""
              }`}
            >
              <HelpCircle className="size-4" />
            </a>
          </Tooltip>
        ) : null}
        <Tooltip tooltipContent="Toggle sidebar" position={isSidebarCollapsed ? "right" : "top"} className="ml-4">
          <button
            type="button"
            aria-label="Toggle sidebar"
            className={`grid place-items-center rounded-md p-1.5 text-secondary outline-none hover:bg-layer-1-hover hover:text-primary ${
              isSidebarCollapsed ? "w-full" : ""
            }`}
            onClick={() => toggleSidebar(!isSidebarCollapsed)}
          >
            <MoveLeft className={`size-4 duration-300 ${isSidebarCollapsed ? "rotate-180" : ""}`} />
          </button>
        </Tooltip>
      </div>
    </div>
  );
});
