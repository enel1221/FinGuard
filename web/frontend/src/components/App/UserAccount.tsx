import { useEffect, useState } from 'react';
import {
  Box, Typography, Card, CardContent, Chip, Avatar, Skeleton, Divider,
} from '@mui/material';
import Grid from '@mui/material/Grid2';
import { Person, Group, Email, Badge } from '@mui/icons-material';
import { api, type UserInfo } from '../../lib/api';

function getInitials(name: string): string {
  return name
    .split(/\s+/)
    .slice(0, 2)
    .map(w => w[0]?.toUpperCase() ?? '')
    .join('');
}

export default function UserAccount() {
  const [user, setUser] = useState<UserInfo | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api.get<UserInfo>('/me')
      .then(setUser)
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  if (loading) {
    return (
      <Box>
        <Typography variant="h5" fontWeight={600} mb={3}>My Account</Typography>
        <Card><CardContent><Skeleton height={200} /></CardContent></Card>
      </Box>
    );
  }

  if (!user) {
    return (
      <Box>
        <Typography variant="h5" fontWeight={600} mb={3}>My Account</Typography>
        <Card>
          <CardContent>
            <Typography color="text.secondary">Unable to load account information.</Typography>
          </CardContent>
        </Card>
      </Box>
    );
  }

  return (
    <Box>
      <Typography variant="h5" fontWeight={600} mb={3}>My Account</Typography>

      <Grid container spacing={3}>
        <Grid size={{ xs: 12, md: 4 }}>
          <Card sx={{ height: '100%' }}>
            <CardContent sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center', py: 4 }}>
              <Avatar
                sx={{
                  width: 80,
                  height: 80,
                  fontSize: '1.8rem',
                  bgcolor: 'primary.main',
                  color: 'primary.contrastText',
                  mb: 2,
                }}
              >
                {getInitials(user.displayName || user.email)}
              </Avatar>
              <Typography variant="h6" fontWeight={600}>
                {user.displayName || user.email}
              </Typography>
              <Typography variant="body2" color="text.secondary">
                {user.email}
              </Typography>
            </CardContent>
          </Card>
        </Grid>

        <Grid size={{ xs: 12, md: 8 }}>
          <Card sx={{ mb: 3 }}>
            <CardContent>
              <Typography variant="subtitle1" fontWeight={600} mb={2}>
                Profile Details
              </Typography>
              <Divider sx={{ mb: 2 }} />

              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, mb: 2 }}>
                <Badge sx={{ color: 'text.secondary' }} />
                <Box>
                  <Typography variant="caption" color="text.secondary">User ID</Typography>
                  <Typography variant="body2" fontFamily="monospace">{user.userId}</Typography>
                </Box>
              </Box>

              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, mb: 2 }}>
                <Person sx={{ color: 'text.secondary' }} />
                <Box>
                  <Typography variant="caption" color="text.secondary">Display Name</Typography>
                  <Typography variant="body2">{user.displayName || '--'}</Typography>
                </Box>
              </Box>

              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
                <Email sx={{ color: 'text.secondary' }} />
                <Box>
                  <Typography variant="caption" color="text.secondary">Email</Typography>
                  <Typography variant="body2">{user.email}</Typography>
                </Box>
              </Box>
            </CardContent>
          </Card>

          <Card>
            <CardContent>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 2 }}>
                <Group sx={{ color: 'text.secondary' }} />
                <Typography variant="subtitle1" fontWeight={600}>Groups</Typography>
              </Box>
              <Divider sx={{ mb: 2 }} />

              {user.groups && user.groups.length > 0 ? (
                <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1 }}>
                  {user.groups.map(g => (
                    <Chip key={g} label={g} variant="outlined" color="primary" />
                  ))}
                </Box>
              ) : (
                <Typography variant="body2" color="text.secondary">
                  Not a member of any groups.
                </Typography>
              )}
            </CardContent>
          </Card>
        </Grid>
      </Grid>
    </Box>
  );
}
