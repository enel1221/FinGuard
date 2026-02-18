import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Box, Typography, Card, CardContent, CardActionArea, Button, TextField, Dialog,
  DialogTitle, DialogContent, DialogActions, Chip,
} from '@mui/material';
import Grid from '@mui/material/Grid2';
import { Add, Folder } from '@mui/icons-material';
import { api, type Project } from '../../lib/api';

export default function ProjectManager() {
  const [projects, setProjects] = useState<Project[]>([]);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [form, setForm] = useState({ name: '', description: '' });
  const navigate = useNavigate();

  const loadProjects = () => {
    api.get<{ projects: Project[] }>('/projects').then(d => setProjects(d.projects || []));
  };

  useEffect(loadProjects, []);

  const openCreate = () => {
    setForm({ name: '', description: '' });
    setDialogOpen(true);
  };

  const handleSave = async () => {
    await api.post('/projects', form);
    setDialogOpen(false);
    loadProjects();
  };

  return (
    <Box>
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={3}>
        <Typography variant="h5" fontWeight={600}>Projects</Typography>
        <Button variant="contained" startIcon={<Add />} onClick={openCreate}>New Project</Button>
      </Box>

      {projects.length === 0 ? (
        <Card>
          <CardContent sx={{ textAlign: 'center', py: 6 }}>
            <Folder sx={{ fontSize: 48, color: 'text.disabled', mb: 1 }} />
            <Typography variant="body1" color="text.secondary" mb={1}>
              No projects yet
            </Typography>
            <Typography variant="body2" color="text.secondary" mb={2}>
              Create your first project to start tracking costs across
              AWS, Azure, GCP, and Kubernetes.
            </Typography>
            <Button variant="outlined" startIcon={<Add />} onClick={openCreate}>
              Create Project
            </Button>
          </CardContent>
        </Card>
      ) : (
        <Grid container spacing={2}>
          {projects.map(p => (
            <Grid key={p.id} size={{ xs: 12, sm: 6, md: 4 }}>
              <Card
                sx={{
                  height: '100%',
                  transition: 'box-shadow 0.2s, transform 0.2s',
                  '&:hover': {
                    boxShadow: 6,
                    transform: 'translateY(-2px)',
                  },
                }}
              >
                <CardActionArea
                  onClick={() => navigate(`/projects/${p.id}`)}
                  sx={{ height: '100%', display: 'flex', flexDirection: 'column', alignItems: 'stretch' }}
                >
                  <CardContent sx={{ flex: 1 }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1.5 }}>
                      <Folder color="primary" />
                      <Typography variant="h6" fontWeight={600} noWrap>
                        {p.name}
                      </Typography>
                    </Box>
                    <Typography
                      variant="body2"
                      color="text.secondary"
                      sx={{
                        mb: 2,
                        display: '-webkit-box',
                        WebkitLineClamp: 2,
                        WebkitBoxOrient: 'vertical',
                        overflow: 'hidden',
                        minHeight: '2.5em',
                      }}
                    >
                      {p.description || 'No description'}
                    </Typography>
                    <Chip
                      label={`Created ${new Date(p.createdAt).toLocaleDateString()}`}
                      size="small"
                      variant="outlined"
                    />
                  </CardContent>
                </CardActionArea>
              </Card>
            </Grid>
          ))}
        </Grid>
      )}

      <Dialog open={dialogOpen} onClose={() => setDialogOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>Create Project</DialogTitle>
        <DialogContent>
          <TextField fullWidth label="Name" value={form.name} sx={{ mt: 1, mb: 2 }}
            onChange={e => setForm(f => ({ ...f, name: e.target.value }))} />
          <TextField fullWidth label="Description" multiline rows={3} value={form.description}
            onChange={e => setForm(f => ({ ...f, description: e.target.value }))} />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDialogOpen(false)}>Cancel</Button>
          <Button variant="contained" onClick={handleSave} disabled={!form.name}>
            Create
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}
