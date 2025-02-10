'use client';

import React, { useState, useEffect } from 'react';
import { useParams } from 'next/navigation'
import Link from 'next/link';

interface Transaction {
  id: string;
  account_id: string;
  posted: number;       // Unix epoch timestamp (in seconds)
  amount: number;
  description: string;
}

export default function AccountPage() {
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const params = useParams();
  useEffect(() => {
    if (params) {
      const id = params.id;
      fetch(`/api/accounts/${id}/transactions`)
        .then(res => res.json())
        .then(setTransactions);
    }
  }, [params]);

  return (
    <div className="container mx-auto px-4 py-8">
      <Link
        href="/accounts"
        className="inline-block mb-4 bg-cyan-500 text-white px-4 py-2 rounded-full hover:bg-cyan-600 transition-colors"
      >
        ← Back to Accounts
      </Link>
      <h1 className="text-2xl font-bold mb-6">Transactions</h1>
      {transactions && transactions.length > 0 ? (
        <div className="overflow-x-auto bg-white shadow-md rounded-lg">
          <table className="min-w-full">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                  Description
                </th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">
                  Amount
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                  Date
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200">
              {transactions.map((txn) => (
                <tr key={txn.id}>
                  <td className="px-6 py-4 whitespace-nowrap">
                    {txn.description}<br />
                    <small className="text-gray-500">
                      {txn.id}
                    </small>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-right">
                    {new Intl.NumberFormat('en-CA', { style: 'currency', currency: 'CAD' }).format(-txn.amount)}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    {new Date(txn.posted * 1000).toLocaleDateString()}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <p>No transactions found for this account.</p>
      )}
    </div>
  );
}
