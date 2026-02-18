import { useEffect, useState } from 'react';
import { Box, Typography, Card, CardContent, Select, MenuItem, FormControl, InputLabel } from '@mui/material';
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend } from 'recharts';
import { api, type Project } from '../../lib/api';

export default function CostExplorer() {
  const [projects, setProjects] = useState<Project[]>([]);
  const [selectedProject, setSelectedProject] = useState('');

  useEffect(() => {
    api.get<{ projects: Project[] }>('/projects').then(data => {
      const p = data.projects || [];
      setProjects(p);
      if (p.length > 0) setSelectedProject(p[0].id);
    });
  }, []);

  const mockData = [
    { name: 'Compute', aws: 450, azure: 320, gcp: 280 },
    { name: 'Storage', aws: 120, azure: 80, gcp: 65 },
    { name: 'Network', aws: 85, azure: 45, gcp: 55 },
    { name: 'Database', aws: 200, azure: 150, gcp: 120 },
    { name: 'Other', aws: 35, azure: 25, gcp: 30 },
  ];

  return (
    <Box>
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={3}>
        <Typography variant="h5" fontWeight={600}>Cost Explorer</Typography>
        <FormControl size="small" sx={{ minWidth: 200 }}>
          <InputLabel>Project</InputLabel>
          <Select value={selectedProject} label="Project"
            onChange={e => setSelectedProject(e.target.value)}>
            {projects.map(p => <MenuItem key={p.id} value={p.id}>{p.name}</MenuItem>)}
          </Select>
        </FormControl>
      </Box>

      <Card sx={{ mb: 2 }}>
        <CardContent>
          <Typography variant="h6" mb={2}>Cost by Category & Provider</Typography>
          <ResponsiveContainer width="100%" height={350}>
            <BarChart data={mockData}>
              <CartesianGrid strokeDasharray="3 3" stroke="#2d3140" />
              <XAxis dataKey="name" stroke="#8b8fa3" />
              <YAxis stroke="#8b8fa3" tickFormatter={v => `$${v}`} />
              <Tooltip
                contentStyle={{ backgroundColor: '#1a1d27', border: '1px solid #2d3140' }}
                formatter={(value: number) => [`$${value}`, '']}
              />
              <Legend />
              <Bar dataKey="aws" fill="#ff9900" name="AWS" radius={[4, 4, 0, 0]} />
              <Bar dataKey="azure" fill="#0078d4" name="Azure" radius={[4, 4, 0, 0]} />
              <Bar dataKey="gcp" fill="#4285f4" name="GCP" radius={[4, 4, 0, 0]} />
            </BarChart>
          </ResponsiveContainer>
        </CardContent>
      </Card>
    </Box>
  );
}
