import { useEffect, useState } from 'react';
import {
  Box, Typography, Card, CardContent, Button, Table, TableHead, TableRow,
  TableCell, TableBody, IconButton, Chip, Dialog, DialogTitle, DialogContent,
  DialogActions, TextField, FormControl, InputLabel, Select, MenuItem,
  ToggleButton, ToggleButtonGroup,
} from '@mui/material';
import { Add, Delete, Person, Group } from '@mui/icons-material';
import {
  api,
  type ProjectRole,
  type SubjectType,
  type RoleType,
} from '../../lib/api';

interface Props {
  projectId: string;
}

export default function ProjectMembers({ projectId }: Props) {
  const [members, setMembers] = useState<ProjectRole[]>([]);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [form, setForm] = useState<{
    subjectType: SubjectType;
    subjectId: string;
    role: RoleType;
  }>({
    subjectType: 'user',
    subjectId: '',
    role: 'viewer',
  });

  const loadMembers = () => {
    api.get<{ members: ProjectRole[] }>(`/projects/${projectId}/members`)
      .then(d => setMembers(d.members || []));
  };

  useEffect(loadMembers, [projectId]);

  const handleAdd = async () => {
    await api.post(`/projects/${projectId}/members`, form);
    setDialogOpen(false);
    setForm({ subjectType: 'user', subjectId: '', role: 'viewer' });
    loadMembers();
  };

  const handleRemove = async (subjectType: SubjectType, subjectId: string) => {
    if (!confirm('Remove this member from the project?')) return;
    await api.delete(`/projects/${projectId}/members/${subjectId}?subjectType=${subjectType}`);
    loadMembers();
  };

  const roleColors: Record<string, 'error' | 'warning' | 'info' | 'default'> = {
    admin: 'error',
    editor: 'warning',
    viewer: 'info',
    'platform-admin': 'error',
  };

  return (
    <Box>
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={2}>
        <Typography variant="subtitle1" fontWeight={600}>Project Members</Typography>
        <Button size="small" variant="contained" startIcon={<Add />}
          onClick={() => setDialogOpen(true)}>
          Add Member
        </Button>
      </Box>

      {members.length === 0 ? (
        <Card variant="outlined">
          <CardContent>
            <Typography variant="body2" color="text.secondary">
              No members assigned. Add users or groups to control access to this project.
            </Typography>
          </CardContent>
        </Card>
      ) : (
        <Card variant="outlined">
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell>Type</TableCell>
                <TableCell>Subject ID</TableCell>
                <TableCell>Role</TableCell>
                <TableCell align="right">Actions</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {members.map(m => (
                <TableRow key={`${m.subjectType}-${m.subjectId}`}>
                  <TableCell>
                    <Chip
                      icon={m.subjectType === 'user' ? <Person /> : <Group />}
                      label={m.subjectType}
                      size="small"
                      variant="outlined"
                    />
                  </TableCell>
                  <TableCell>
                    <Typography variant="body2" fontFamily="monospace">
                      {m.subjectId}
                    </Typography>
                  </TableCell>
                  <TableCell>
                    <Chip
                      label={m.role}
                      size="small"
                      color={roleColors[m.role] || 'default'}
                    />
                  </TableCell>
                  <TableCell align="right">
                    <IconButton
                      size="small"
                      color="error"
                      onClick={() => handleRemove(m.subjectType, m.subjectId)}
                    >
                      <Delete fontSize="small" />
                    </IconButton>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </Card>
      )}

      <Dialog open={dialogOpen} onClose={() => setDialogOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>Add Member</DialogTitle>
        <DialogContent>
          <Box sx={{ mt: 1, mb: 2 }}>
            <Typography variant="body2" color="text.secondary" mb={1}>Subject Type</Typography>
            <ToggleButtonGroup
              exclusive
              value={form.subjectType}
              onChange={(_, v) => v && setForm(f => ({ ...f, subjectType: v }))}
              size="small"
            >
              <ToggleButton value="user">
                <Person sx={{ mr: 0.5 }} fontSize="small" /> User
              </ToggleButton>
              <ToggleButton value="group">
                <Group sx={{ mr: 0.5 }} fontSize="small" /> Group
              </ToggleButton>
            </ToggleButtonGroup>
          </Box>

          <TextField
            fullWidth
            label={form.subjectType === 'user' ? 'User ID' : 'Group ID'}
            value={form.subjectId}
            onChange={e => setForm(f => ({ ...f, subjectId: e.target.value }))}
            sx={{ mb: 2 }}
            helperText={`Enter the ${form.subjectType} ID to grant access`}
          />

          <FormControl fullWidth>
            <InputLabel>Role</InputLabel>
            <Select
              value={form.role}
              label="Role"
              onChange={e => setForm(f => ({ ...f, role: e.target.value as RoleType }))}
            >
              <MenuItem value="viewer">Viewer - Read-only access</MenuItem>
              <MenuItem value="editor">Editor - Can modify resources</MenuItem>
              <MenuItem value="admin">Admin - Full project control</MenuItem>
            </Select>
          </FormControl>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDialogOpen(false)}>Cancel</Button>
          <Button variant="contained" onClick={handleAdd} disabled={!form.subjectId}>
            Add
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}
