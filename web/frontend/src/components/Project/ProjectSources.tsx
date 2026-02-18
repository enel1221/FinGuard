import { useEffect, useState } from 'react';
import {
  Box, Typography, Card, CardContent, Button, Table, TableHead, TableRow,
  TableCell, TableBody, IconButton, Chip,
} from '@mui/material';
import { Add, Delete } from '@mui/icons-material';
import { api, type CostSource } from '../../lib/api';
import AddSourceDialog from './AddSourceDialog';

interface Props {
  projectId: string;
}

const typeLabels: Record<string, string> = {
  aws_account: 'AWS',
  azure_subscription: 'Azure',
  gcp_project: 'GCP',
  kubernetes: 'Kubernetes',
  plugin: 'Plugin',
};

const typeColors: Record<string, 'warning' | 'info' | 'success' | 'secondary' | 'default'> = {
  aws_account: 'warning',
  azure_subscription: 'info',
  gcp_project: 'success',
  kubernetes: 'secondary',
  plugin: 'default',
};

export default function ProjectSources({ projectId }: Props) {
  const [sources, setSources] = useState<CostSource[]>([]);
  const [dialogOpen, setDialogOpen] = useState(false);

  const loadSources = () => {
    api.get<{ sources: CostSource[] }>(`/projects/${projectId}/sources`)
      .then(d => setSources(d.sources || []));
  };

  useEffect(loadSources, [projectId]);

  const handleDelete = async (id: string) => {
    if (!confirm('Delete this cost source? Collected data will also be removed.')) return;
    await api.delete(`/projects/${projectId}/sources/${id}`);
    loadSources();
  };

  return (
    <Box>
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={2}>
        <Typography variant="subtitle1" fontWeight={600}>Cost Sources</Typography>
        <Button size="small" variant="contained" startIcon={<Add />}
          onClick={() => setDialogOpen(true)}>
          Add Source
        </Button>
      </Box>

      {sources.length === 0 ? (
        <Card variant="outlined">
          <CardContent>
            <Typography variant="body2" color="text.secondary">
              No cost sources configured. Add a source to start collecting cost data for this project.
            </Typography>
          </CardContent>
        </Card>
      ) : (
        <Card variant="outlined">
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell>Name</TableCell>
                <TableCell>Type</TableCell>
                <TableCell>Status</TableCell>
                <TableCell>Last Collected</TableCell>
                <TableCell align="right">Actions</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {sources.map(src => (
                <TableRow key={src.id}>
                  <TableCell>
                    <Typography variant="body2" fontWeight={500}>{src.name}</Typography>
                  </TableCell>
                  <TableCell>
                    <Chip
                      label={typeLabels[src.type] || src.type}
                      size="small"
                      color={typeColors[src.type] || 'default'}
                    />
                  </TableCell>
                  <TableCell>
                    <Chip
                      label={src.enabled ? 'Enabled' : 'Disabled'}
                      size="small"
                      color={src.enabled ? 'success' : 'default'}
                      variant="outlined"
                    />
                  </TableCell>
                  <TableCell>
                    <Typography variant="body2" color="text.secondary">
                      {src.lastCollectedAt
                        ? new Date(src.lastCollectedAt).toLocaleString()
                        : 'Never'}
                    </Typography>
                  </TableCell>
                  <TableCell align="right">
                    <IconButton size="small" color="error" onClick={() => handleDelete(src.id)}>
                      <Delete fontSize="small" />
                    </IconButton>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </Card>
      )}

      <AddSourceDialog
        projectId={projectId}
        open={dialogOpen}
        onClose={() => setDialogOpen(false)}
        onCreated={loadSources}
      />
    </Box>
  );
}
