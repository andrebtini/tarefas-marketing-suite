/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";
import Link from "next/link";
import { EAuthModes, MS_URL_PRIVACIDADE, MS_URL_TERMOS } from "@plane/constants";

interface TermsAndConditionsProps {
  authType?: EAuthModes;
}

// MS: termos e privacidade do site da Marketing Suite (MRC-004)
const LEGAL_LINKS = {
  termsOfService: MS_URL_TERMOS,
  privacyPolicy: MS_URL_PRIVACIDADE,
} as const;

const MESSAGES = {
  [EAuthModes.SIGN_UP]: "By creating an account",
  [EAuthModes.SIGN_IN]: "By signing in",
} as const;

// Reusable link component to reduce duplication
function LegalLink({ href, children }: { href: string; children: React.ReactNode }) {
  return (
    <Link href={href} className="text-secondary" target="_blank" rel="noopener noreferrer">
      <span className="text-13 font-medium underline hover:cursor-pointer">{children}</span>
    </Link>
  );
}

export function TermsAndConditions({ authType = EAuthModes.SIGN_IN }: TermsAndConditionsProps) {
  return (
    <div className="flex items-center justify-center">
      <p className="text-center text-13 whitespace-pre-line text-tertiary">
        {`${MESSAGES[authType]}, you understand and agree to \n our `}
        <LegalLink href={LEGAL_LINKS.termsOfService}>Terms of Service</LegalLink> and{" "}
        <LegalLink href={LEGAL_LINKS.privacyPolicy}>Privacy Policy</LegalLink>.
      </p>
    </div>
  );
}
