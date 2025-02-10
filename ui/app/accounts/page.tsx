'use client'

import React, { useState, useEffect } from 'react';
import Link from 'next/link';
// Define the TypeScript type for an account (aligned with the Rust model from accounts.rs)
interface Account {
  id: string;
  name: string;
  currency: string;
  balance: number;
  available_balance: number;
  balance_date: number;
  organization_id: string;
  extra?: any; // eslint-disable-line
}

// This is a server component by default in Next.js App Router.
// We fetch the accounts on the server side before rendering.
export default function AccountsPage() {

  const [accounts, setAccounts] = useState<Account[]>([]);

  useEffect(() => {
    fetch('/api/accounts')
      .then(res => res.json())
      .then(setAccounts);
  }, []);

  return (
    <div className="min-h-screen bg-gray-100 p-4 flex justify-center">
      <div className="w-full max-w-2xl p-4">
        <h1 className="text-3xl font-bold text-gray-800 mb-4 text-center">Accounts</h1>
        <ul className="grid gap-2">
          {accounts.map((account) => (
            <li
              key={account.id}
              className="bg-white p-4 rounded-xl shadow transition-shadow hover:shadow-lg"
            >
              <Link href={`/accounts/${account.id}`}>
                <div className="flex flex-col md:flex-row justify-between items-center">
                  <div>
                    <h2 className="text-lg font-semibold text-gray-900">
                      {account.name}
                    </h2>
                    <p className="text-gray-600 mb-1">
                      Currency: {account.currency}
                    </p>
                  </div>
                  <div className="mt-2 md:mt-0 text-right">
                    <p className="text-gray-700">
                      Balance: {new Intl.NumberFormat('en-US', { style: 'currency', currency: account.currency }).format(account.balance)}
                    </p>
                    <p className="text-gray-500 text-xs">
                      {new Date(account.balance_date * 1000).toLocaleDateString()}
                    </p>
                  </div>
                </div>
                </Link>
            </li>
          ))}
        </ul>
      </div>
    </div>
  );
}
