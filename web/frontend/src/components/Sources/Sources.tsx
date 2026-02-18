import { useEffect, useState } from 'react';
import {
  Box, Typography, Card, CardContent, Button, Table, TableHead, TableRow,
  TableCell, TableBody, Chip, FormControl, InputLabel, Select, MenuItem,
} from '@mui/material';
import { Add } from '@mui/icons-material';
import { api, type Project, type CostSource } from '../../lib/api';
import AddSourceDialog from '../Project/AddSourceDialog';

export default function Sources() {
  const [projects, setProjects] = useState<Project[]>([]);
  const [sources, setSources] = useState<Record<string, CostSource[]>>({});
  const [dialogOpen, setDialogOpen] = useState(false);
  const [selectedProject, setSelectedProject] = useState('');

  const loadAll = () => {
    api.get<{ projects: Project[] }>('/projects').then(data => {
      const list = data.projects || [];
      setProjects(list);
      if (list.length > 0 && !selectedProject) {
        setSelectedProject(list[0].id);
      }
      list.forEach(p => {
        api.get<{ sources: CostSource[] }>(`/projects/${p.id}/sources`).then(s => {
          setSources(prev => ({ ...prev, [p.id]: s.sources || [] }));
        });
      });
    });
  };

  useEffect(loadAll, []);

  const reloadSources = () => {
    if (!selectedProject) return;
    api.get<{ sources: CostSource[] }>(`/projects/${selectedProject}/sources`).then(s => {
      setSources(prev => ({ ...prev, [selectedProject]: s.sources || [] }));
    });
  };

  const typeColors: Record<string, 'warning' | 'info' | 'success' | 'secondary' | 'default'> = {
    aws_account: 'warning',
    azure_subscription: 'info',
    gcp_project: 'success',
    kubernetes: 'secondary',
    plugin: 'default',
  };

  const typeLabels: Record<string, string> = {
    aws_account: 'AWS',
    azure_subscription: 'Azure',
    gcp_project: 'GCP',
    kubernetes: 'Kubernetes',
    plugin: 'Plugin',
  };

  return (
    <Box>
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={3}>
        <Typography variant="h5" fontWeight={600}>Cost Sources</Typography>
        <Box display="flex" gap={2} alignItems="center">
          {projects.length > 0 && (
            <FormControl size="small" sx={{ minWidth: 180 }}>
              <InputLabel>Project</InputLabel>
              <Select
                value={selectedProject}
                label="Project"
                onChange={e => setSelectedProject(e.target.value)}
              >
                {projects.map(p => (
                  <MenuItem key={p.id} value={p.id}>{p.name}</MenuItem>
                ))}
              </Select>
            </FormControl>
          )}
          <Button
            variant="contained"
            startIcon={<Add />}
            onClick={() => setDialogOpen(true)}
            disabled={!selectedProject}
          >
            Add Source
          </Button>
        </Box>
      </Box>

      {projects.length === 0 ? (
        <Card>
          <CardContent>
            <Typography variant="body2" color="text.secondary">
              No projects yet. Create a project first to add cost sources.
            </Typography>
          </CardContent>
        </Card>
      ) : (
        projects.map(project => (
          <Card key={project.id} sx={{ mb: 2 }}>
            <CardContent>
              <Typography variant="h6" mb={1}>{project.name}</Typography>
              {(sources[project.id] || []).length === 0 ? (
                <Typography variant="body2" color="text.secondary">No cost sources configured</Typography>
              ) : (
                <Table size="small">
                  <TableHead>
                    <TableRow>
                      <TableCell>Name</TableCell>
                      <TableCell>Type</TableCell>
                      <TableCell>Status</TableCell>
                      <TableCell>Last Collected</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {(sources[project.id] || []).map(src => (
                      <TableRow key={src.id}>
                        <TableCell>{src.name}</TableCell>
                        <TableCell>
                          <Chip label={typeLabels[src.type] || src.type} size="small" color={typeColors[src.type] || 'default'} />
                        </TableCell>
                        <TableCell>
                          <Chip label={src.enabled ? 'Enabled' : 'Disabled'} size="small"
                            color={src.enabled ? 'success' : 'default'} variant="outlined" />
                        </TableCell>
                        <TableCell>
                          {src.lastCollectedAt
                            ? new Date(src.lastCollectedAt).toLocaleString()
                            : 'Never'}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              )}
            </CardContent>
          </Card>
        ))
      )}

      {selectedProject && (
        <AddSourceDialog
          projectId={selectedProject}
          open={dialogOpen}
          onClose={() => setDialogOpen(false)}
          onCreated={reloadSources}
        />
      )}
    </Box>
  );
}
