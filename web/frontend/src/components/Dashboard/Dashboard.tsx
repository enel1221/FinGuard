import { useEffect, useState } from 'react';
import {
  Box, Card, CardContent, Typography, Chip, Table, TableHead, TableRow,
  TableCell, TableBody, CircularProgress,
} from '@mui/material';
import Grid from '@mui/material/Grid2';
import { api, type HealthResponse, type CostSummary } from '../../lib/api';

export default function Dashboard() {
  const [health, setHealth] = useState<HealthResponse | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    Promise.all([
      api.get<HealthResponse>('/health').catch(() => null),
    ]).then(([h]) => {
      setHealth(h);
      setLoading(false);
    });
  }, []);

  if (loading) {
    return (
      <Box display="flex" justifyContent="center" mt={8}>
        <CircularProgress color="primary" />
      </Box>
    );
  }

  return (
    <Box>
      <Typography variant="h5" fontWeight={600} mb={3}>Dashboard</Typography>
      <Grid container spacing={2} mb={3}>
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <Card>
            <CardContent>
              <Typography variant="caption" color="text.secondary">OpenCost</Typography>
              <Box mt={0.5}>
                <Chip
                  label={health?.services?.opencost || 'unknown'}
                  color={health?.services?.opencost === 'healthy' ? 'success' : 'warning'}
                  size="small"
                />
              </Box>
            </CardContent>
          </Card>
        </Grid>
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <Card>
            <CardContent>
              <Typography variant="caption" color="text.secondary">Cluster Cache</Typography>
              <Box mt={0.5}>
                <Chip
                  label={health?.services?.cluster_cache || 'unavailable'}
                  color={health?.services?.cluster_cache === 'ready' ? 'success' : 'default'}
                  size="small"
                />
              </Box>
            </CardContent>
          </Card>
        </Grid>
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <Card>
            <CardContent>
              <Typography variant="caption" color="text.secondary">Status</Typography>
              <Typography variant="h4" fontWeight={700} color="primary">
                {health?.status || '--'}
              </Typography>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      <Card>
        <CardContent>
          <Typography variant="h6" mb={2}>Getting Started</Typography>
          <Typography variant="body2" color="text.secondary">
            Create a <strong>Project</strong> to get started, then add <strong>Cost Sources</strong> (AWS accounts,
            Azure subscriptions, GCP projects, or Kubernetes clusters) to begin tracking costs.
          </Typography>
        </CardContent>
      </Card>
    </Box>
  );
}
