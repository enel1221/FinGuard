import { useEffect, useState } from 'react';
import {
  Box, Typography, Card, CardContent, Divider, Button, TextField,
  Dialog, DialogTitle, DialogContent, DialogActions, Skeleton,
} from '@mui/material';
import Grid from '@mui/material/Grid2';
import { Edit, CalendarToday, AttachMoney } from '@mui/icons-material';
import { api, type Project, type CostSummary } from '../../lib/api';

interface Props {
  project: Project;
  onUpdated: () => void;
}

export default function ProjectOverview({ project, onUpdated }: Props) {
  const [costs, setCosts] = useState<CostSummary | null>(null);
  const [costsLoading, setCostsLoading] = useState(true);
  const [editOpen, setEditOpen] = useState(false);
  const [form, setForm] = useState({ name: project.name, description: project.description });

  useEffect(() => {
    setCostsLoading(true);
    api.get<CostSummary>(`/projects/${project.id}/costs`)
      .then(setCosts)
      .catch(() => {})
      .finally(() => setCostsLoading(false));
  }, [project.id]);

  useEffect(() => {
    setForm({ name: project.name, description: project.description });
  }, [project.name, project.description]);

  const handleSave = async () => {
    await api.put(`/projects/${project.id}`, form);
    setEditOpen(false);
    onUpdated();
  };

  const fmt = (n: number) =>
    new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' }).format(n);

  return (
    <Box>
      <Box display="flex" justifyContent="space-between" alignItems="flex-start" mb={3}>
        <Box>
          <Typography variant="body1" color="text.secondary" mb={0.5}>
            {project.description || 'No description'}
          </Typography>
        </Box>
        <Button size="small" startIcon={<Edit />} onClick={() => setEditOpen(true)}>
          Edit
        </Button>
      </Box>

      <Grid container spacing={2} mb={3}>
        <Grid size={{ xs: 12, sm: 6 }}>
          <Card variant="outlined">
            <CardContent>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
                <CalendarToday fontSize="small" color="action" />
                <Typography variant="caption" color="text.secondary">Created</Typography>
              </Box>
              <Typography variant="body1">
                {new Date(project.createdAt).toLocaleDateString('en-US', {
                  year: 'numeric', month: 'long', day: 'numeric',
                })}
              </Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid size={{ xs: 12, sm: 6 }}>
          <Card variant="outlined">
            <CardContent>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
                <CalendarToday fontSize="small" color="action" />
                <Typography variant="caption" color="text.secondary">Last Updated</Typography>
              </Box>
              <Typography variant="body1">
                {new Date(project.updatedAt).toLocaleDateString('en-US', {
                  year: 'numeric', month: 'long', day: 'numeric',
                })}
              </Typography>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      <Typography variant="subtitle1" fontWeight={600} mb={2}>Cost Summary</Typography>
      <Divider sx={{ mb: 2 }} />

      {costsLoading ? (
        <Grid container spacing={2}>
          {[0, 1, 2, 3].map(i => (
            <Grid key={i} size={{ xs: 6, md: 3 }}>
              <Skeleton variant="rounded" height={80} />
            </Grid>
          ))}
        </Grid>
      ) : costs && costs.recordCount > 0 ? (
        <Grid container spacing={2}>
          {[
            { label: 'List Cost', value: fmt(costs.totalListCost) },
            { label: 'Net Cost', value: fmt(costs.totalNetCost) },
            { label: 'Amortized', value: fmt(costs.totalAmortized) },
            { label: 'Records', value: costs.recordCount.toLocaleString() },
          ].map(item => (
            <Grid key={item.label} size={{ xs: 6, md: 3 }}>
              <Card variant="outlined">
                <CardContent>
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5, mb: 0.5 }}>
                    <AttachMoney fontSize="small" color="action" />
                    <Typography variant="caption" color="text.secondary">{item.label}</Typography>
                  </Box>
                  <Typography variant="h6" fontWeight={600}>{item.value}</Typography>
                </CardContent>
              </Card>
            </Grid>
          ))}
        </Grid>
      ) : (
        <Card variant="outlined">
          <CardContent>
            <Typography variant="body2" color="text.secondary">
              No cost data collected yet. Add cost sources to start tracking.
            </Typography>
          </CardContent>
        </Card>
      )}

      <Dialog open={editOpen} onClose={() => setEditOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>Edit Project</DialogTitle>
        <DialogContent>
          <TextField fullWidth label="Name" value={form.name} sx={{ mt: 1, mb: 2 }}
            onChange={e => setForm(f => ({ ...f, name: e.target.value }))} />
          <TextField fullWidth label="Description" multiline rows={3} value={form.description}
            onChange={e => setForm(f => ({ ...f, description: e.target.value }))} />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setEditOpen(false)}>Cancel</Button>
          <Button variant="contained" onClick={handleSave} disabled={!form.name}>Save</Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}
