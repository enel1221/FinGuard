import { useEffect, useState } from 'react';
import {
  Box, Typography, Card, CardContent, Table, TableHead, TableRow, TableCell, TableBody, Chip,
} from '@mui/material';
import { api, type PluginMeta } from '../../lib/api';

export default function PluginsPage() {
  const [plugins, setPlugins] = useState<PluginMeta[]>([]);

  useEffect(() => {
    api.get<{ plugins: PluginMeta[] }>('/plugins')
      .then(data => setPlugins(data.plugins || []))
      .catch(() => {});
  }, []);

  return (
    <Box>
      <Typography variant="h5" fontWeight={600} mb={3}>Plugins</Typography>
      <Card>
        <CardContent>
          {plugins.length === 0 ? (
            <Typography variant="body2" color="text.secondary">No plugins registered.</Typography>
          ) : (
            <Table size="small">
              <TableHead>
                <TableRow>
                  <TableCell>Name</TableCell>
                  <TableCell>Version</TableCell>
                  <TableCell>Type</TableCell>
                  <TableCell>Description</TableCell>
                  <TableCell>Topics</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {plugins.map(p => (
                  <TableRow key={p.name}>
                    <TableCell>{p.name}</TableCell>
                    <TableCell>{p.version}</TableCell>
                    <TableCell><Chip label={p.type} size="small" color="primary" variant="outlined" /></TableCell>
                    <TableCell>{p.description}</TableCell>
                    <TableCell>{(p.topics || []).join(', ')}</TableCell>
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
