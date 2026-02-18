import { useEffect, useState } from 'react';
import {
  Box, Typography, Card, CardContent, Table, TableHead, TableRow, TableCell, TableBody, Chip,
} from '@mui/material';
import { api } from '../../lib/api';

interface Recommendation {
  namespace: string;
  resourceType: string;
  requested: number;
  used: number;
  estimatedSavings: number;
  severity: string;
}

export default function Recommendations() {
  const [recs, setRecs] = useState<Recommendation[]>([]);

  useEffect(() => {
    api.get<{ recommendations: Recommendation[] }>('/plugins/costbreakdown/recommendations')
      .then(data => setRecs(data.recommendations || []))
      .catch(() => {});
  }, []);

  return (
    <Box>
      <Typography variant="h5" fontWeight={600} mb={3}>Recommendations</Typography>
      <Card>
        <CardContent>
          {recs.length === 0 ? (
            <Typography variant="body2" color="text.secondary">
              No recommendations yet. Waiting for cost data collection.
            </Typography>
          ) : (
            <Table size="small">
              <TableHead>
                <TableRow>
                  <TableCell>Namespace</TableCell>
                  <TableCell>Resource</TableCell>
                  <TableCell>Requested</TableCell>
                  <TableCell>Used</TableCell>
                  <TableCell>Est. Savings</TableCell>
                  <TableCell>Severity</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {recs.map((r, i) => (
                  <TableRow key={i}>
                    <TableCell>{r.namespace}</TableCell>
                    <TableCell>{r.resourceType}</TableCell>
                    <TableCell>{r.requested.toFixed(3)}</TableCell>
                    <TableCell>{r.used.toFixed(3)}</TableCell>
                    <TableCell>${r.estimatedSavings.toFixed(2)}</TableCell>
                    <TableCell>
                      <Chip label={r.severity} size="small"
                        color={r.severity === 'high' ? 'error' : r.severity === 'medium' ? 'warning' : 'success'} />
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </Box>
  );
}
