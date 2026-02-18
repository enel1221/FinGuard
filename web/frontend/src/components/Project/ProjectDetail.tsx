import { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import {
  Box, Typography, Tabs, Tab, Breadcrumbs, Link, Skeleton, IconButton,
} from '@mui/material';
import { ArrowBack, Folder } from '@mui/icons-material';
import { api, type Project } from '../../lib/api';
import ProjectOverview from './ProjectOverview';
import ProjectMembers from './ProjectMembers';
import ProjectSources from './ProjectSources';

export default function ProjectDetail() {
  const { projectId } = useParams<{ projectId: string }>();
  const navigate = useNavigate();
  const [project, setProject] = useState<Project | null>(null);
  const [loading, setLoading] = useState(true);
  const [tab, setTab] = useState(0);

  const loadProject = () => {
    if (!projectId) return;
    setLoading(true);
    api.get<Project>(`/projects/${projectId}`)
      .then(setProject)
      .catch(() => setProject(null))
      .finally(() => setLoading(false));
  };

  useEffect(loadProject, [projectId]);

  if (loading) {
    return (
      <Box>
        <Skeleton width={300} height={32} sx={{ mb: 2 }} />
        <Skeleton width="100%" height={48} sx={{ mb: 2 }} />
        <Skeleton variant="rounded" height={300} />
      </Box>
    );
  }

  if (!project) {
    return (
      <Box>
        <Typography variant="h5" mb={2}>Project not found</Typography>
        <Typography variant="body2" color="text.secondary" mb={2}>
          The requested project does not exist or you don't have access.
        </Typography>
        <Link component="button" onClick={() => navigate('/projects')}>
          Back to Projects
        </Link>
      </Box>
    );
  }

  return (
    <Box>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
        <IconButton size="small" onClick={() => navigate('/projects')}>
          <ArrowBack />
        </IconButton>
        <Breadcrumbs>
          <Link
            component="button"
            underline="hover"
            color="inherit"
            onClick={() => navigate('/projects')}
            sx={{ cursor: 'pointer' }}
          >
            Projects
          </Link>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
            <Folder fontSize="small" color="primary" />
            <Typography color="text.primary" fontWeight={600}>{project.name}</Typography>
          </Box>
        </Breadcrumbs>
      </Box>

      <Tabs
        value={tab}
        onChange={(_, v) => setTab(v)}
        sx={{ borderBottom: 1, borderColor: 'divider', mb: 3 }}
      >
        <Tab label="Overview" />
        <Tab label="Members" />
        <Tab label="Sources" />
      </Tabs>

      {tab === 0 && <ProjectOverview project={project} onUpdated={loadProject} />}
      {tab === 1 && <ProjectMembers projectId={project.id} />}
      {tab === 2 && <ProjectSources projectId={project.id} />}
    </Box>
  );
}
