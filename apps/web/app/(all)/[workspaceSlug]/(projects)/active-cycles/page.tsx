/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Navigate, useParams } from "react-router";
// MS: a página só vendia o plano pago (MRC-001); volta para o Início do espaço
export default function WorkspaceActiveCyclesPage() {
  const { workspaceSlug } = useParams();
  return <Navigate to={`/${workspaceSlug}`} replace />;
}
